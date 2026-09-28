package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/ratelimit"
	"golang.org/x/time/rate"
)

// maxSessionRequestBytes는 토큰 갱신·로그아웃 요청 본문의 최대 크기입니다
const maxSessionRequestBytes = 4 << 10 // 4KB

// 토큰 갱신 요청 제한: 계열(로그인 세션)당 분당 10회
const (
	refreshRateInterval = time.Minute / 10
	refreshRateBurst    = 10
)

// rate limiter 정리 주기와 idle 기준
// 분당 10회·burst 10이면 토큰 버킷이 가득 차는 데 최대 1분이 걸리므로,
// idle 30분은 버킷이 가득 찬 뒤로도 충분히 여유가 있어 정리가 제한을 느슨하게 만들지 않습니다
const (
	rateLimiterCleanupInterval = 10 * time.Minute
	rateLimiterCleanupIdle     = 30 * time.Minute
)

// RefreshTokenRequest는 토큰 갱신과 로그아웃 요청 본문입니다
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"ohrt_..."`
}

// SessionHandler는 Refresh Token으로 세션을 연장하거나 폐기하는 요청을 처리합니다
type SessionHandler struct {
	db            database.DBTX
	refreshConfig auth.RefreshTokenConfig
	limiter       *ratelimit.RateLimiter
	// now는 현재 시각을 반환합니다 (테스트에서 시간을 고정하기 위해 교체할 수 있음)
	now func() time.Time
}

// NewSessionHandler는 SessionHandler의 새 인스턴스를 생성합니다
func NewSessionHandler(db database.DBTX) *SessionHandler {
	return &SessionHandler{
		db:            db,
		refreshConfig: auth.LoadRefreshTokenConfig(),
		limiter:       ratelimit.NewRateLimiter(rate.Every(refreshRateInterval), refreshRateBurst),
		now:           time.Now,
	}
}

// StartRateLimiterCleanup은 토큰 갱신 rate limiter에서 오래 쓰이지 않은 키(로그인 계열)를
// 주기적으로 정리하는 고루틴을 시작합니다. ctx가 취소되면 고루틴이 종료됩니다
func (h *SessionHandler) StartRateLimiterCleanup(ctx context.Context) {
	h.limiter.StartCleanup(ctx, rateLimiterCleanupInterval, rateLimiterCleanupIdle)
}

// Refresh는 Refresh Token을 한 번 사용하고 새 Access Token과 Refresh Token을 발급합니다
//
// @Summary      어드민 토큰 갱신
// @Description  Refresh Token을 회전해 새 토큰 쌍을 발급합니다. 사용된 토큰을 유예 시간(30초) 밖에서 다시 제출하면 세션 전체가 폐기됩니다.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshTokenRequest true "Refresh Token"
// @Success      200 {object} TokenPairResponse
// @Header       200 {string} Cache-Control "no-store"
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "INVALID_REFRESH_TOKEN, REFRESH_TOKEN_EXPIRED, REFRESH_TOKEN_REUSED, USER_NOT_FOUND"
// @Failure      429 {object} ErrorResponse "RATE_LIMIT_EXCEEDED"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Router       /auth/refresh [post]
func (h *SessionHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, ok := decodeRefreshTokenRequest(w, r)
	if !ok {
		return
	}

	pair, err := rotateSession(r.Context(), h.db, refreshToken, h.refreshConfig, h.limiter, h.now())
	if err != nil {
		respondRefreshError(w, err)
		return
	}

	// 토큰이 담긴 응답은 어디에도 캐시되지 않게 합니다 (RFC 6749 5.1)
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusOK, pair)
}

// Logout은 Refresh Token이 속한 세션(계열)을 폐기합니다
// 토큰이 없거나 이미 폐기되었어도 204를 반환합니다 (토큰 존재 여부를 알려주지 않기 위함)
//
// @Summary      어드민 로그아웃
// @Description  Refresh Token이 속한 세션을 폐기합니다. 이미 발급된 Access Token은 만료될 때까지 유효합니다.
// @Tags         auth
// @Accept       json
// @Param        request body RefreshTokenRequest true "Refresh Token"
// @Success      204 "No Content"
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Router       /auth/logout [post]
func (h *SessionHandler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken, ok := decodeRefreshTokenRequest(w, r)
	if !ok {
		return
	}

	if err := revokeSession(r.Context(), h.db, refreshToken, h.now()); err != nil {
		log.Printf("[ERROR] failed to revoke session: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to revoke session", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// decodeRefreshTokenRequest는 요청 본문에서 refresh_token을 꺼냅니다
// 본문이 잘못되었으면 400 응답을 쓰고 false를 반환합니다
func decodeRefreshTokenRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req RefreshTokenRequest
	body := http.MaxBytesReader(w, r.Body, maxSessionRequestBytes)
	if err := json.NewDecoder(body).Decode(&req); err != nil || req.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "refresh_token is required", nil)
		return "", false
	}
	return req.RefreshToken, true
}

// respondRefreshError는 세션 규칙 에러를 HTTP 상태와 에러 코드로 바꿔 응답합니다
func respondRefreshError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errSessionInvalid):
		respondError(w, http.StatusUnauthorized, ErrInvalidRefreshToken, "Refresh token is invalid or revoked", nil)
	case errors.Is(err, errSessionExpired):
		respondError(w, http.StatusUnauthorized, ErrRefreshTokenExpired, "Refresh token has expired", nil)
	case errors.Is(err, errSessionReused):
		respondError(w, http.StatusUnauthorized, ErrRefreshTokenReused, "Refresh token was reused; the session has been revoked", nil)
	case errors.Is(err, errSessionUserMissing):
		respondError(w, http.StatusUnauthorized, ErrUserNotFound, "User not found", nil)
	case errors.Is(err, errSessionRateLimited):
		w.Header().Set("Retry-After", strconv.Itoa(int(refreshRateInterval.Seconds())))
		respondError(w, http.StatusTooManyRequests, ErrRateLimitExceeded, "Too many refresh requests", nil)
	default:
		log.Printf("[ERROR] failed to refresh session: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to refresh session", nil)
	}
}
