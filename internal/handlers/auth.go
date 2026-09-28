package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
)

// AuthHandler는 인증 관련 HTTP 요청을 처리합니다
type AuthHandler struct {
	db            *sql.DB
	refreshConfig auth.RefreshTokenConfig
	// now는 현재 시각을 반환합니다 (테스트에서 시간을 고정하기 위해 교체할 수 있음)
	now func() time.Time
	// verifyIDToken은 Google ID Token을 검증합니다 (테스트에서 가짜 검증 결과를 주입하기 위해 교체할 수 있음)
	verifyIDToken func(ctx context.Context, idToken string) (*auth.GoogleIDTokenPayload, error)
}

// NewAuthHandler는 AuthHandler의 새 인스턴스를 생성합니다
func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{
		db:            db,
		refreshConfig: auth.LoadRefreshTokenConfig(),
		now:           time.Now,
		verifyIDToken: auth.VerifyGoogleIDToken,
	}
}

// GoogleVerifyRequest는 Google ID Token 검증 요청 본문입니다
type GoogleVerifyRequest struct {
	IDToken string `json:"id_token"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// GoogleVerifyResponse는 Google ID Token 검증 성공 응답입니다
type GoogleVerifyResponse struct {
	TokenPairResponse

	// Token은 access_token과 같은 값입니다 (token 필드만 읽는 클라이언트 호환용, 전환 3단계에서 제거)
	Token string `json:"token"`

	User *models.User `json:"user"`
}

// GoogleVerify는 Google ID Token을 검증하고 Access Token과 Refresh Token을 발급합니다
//
// @Summary      Google OAuth 인증 및 토큰 발급
// @Description  Google ID Token을 검증하고 사용자를 생성/조회한 후 Access Token과 Refresh Token을 발급합니다
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body GoogleVerifyRequest true "Google 인증 정보"
// @Success      200 {object} GoogleVerifyResponse "토큰 쌍 및 사용자 정보"
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "INVALID_ID_TOKEN"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Router       /auth/google/verify [post]
func (h *AuthHandler) GoogleVerify(w http.ResponseWriter, r *http.Request) {
	// 1. Content-Type 검증
	if r.Header.Get("Content-Type") != "application/json" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Content-Type must be application/json", nil)
		return
	}

	// 2. JSON 요청 파싱
	var req GoogleVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid JSON", nil)
		return
	}

	// 3. 입력 검증
	if req.IDToken == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "id_token is required", nil)
		return
	}
	if req.Email == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "email is required", nil)
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "name is required", nil)
		return
	}

	// 4. Google ID Token 검증
	payload, err := h.verifyIDToken(r.Context(), req.IDToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidIDToken) {
			respondError(w, http.StatusUnauthorized, ErrInvalidIDToken, "Invalid Google ID Token", nil)
			return
		}
		log.Printf("[ERROR] google verify: verify id token: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to verify Google ID Token", nil)
		return
	}

	// 5. 트랜잭션 시작
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("[ERROR] google verify: begin transaction: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to start transaction", nil)
		return
	}
	defer tx.Rollback()

	// 6. Google ID로 사용자 조회
	user, err := database.GetUserByGoogleID(r.Context(), tx, payload.GoogleID)
	if err != nil {
		log.Printf("[ERROR] google verify: get user: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get user", nil)
		return
	}

	// 7. 사용자가 없으면 생성
	if user == nil {
		user = &models.User{
			Email:      req.Email,
			Name:       req.Name,
			PictureURL: req.Picture,
			GoogleID:   payload.GoogleID,
		}

		if err := database.CreateUser(r.Context(), tx, user); err != nil {
			log.Printf("[ERROR] google verify: create user: %v", err)
			respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to create user", nil)
			return
		}
	}

	// 8. 세션 발급 (Access Token + Refresh Token)
	// Refresh Token 저장도 사용자 생성과 같은 트랜잭션에서 처리합니다
	pair, err := issueSession(r.Context(), tx, user, h.refreshConfig, h.now())
	if err != nil {
		log.Printf("[ERROR] google verify: issue session: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to issue session", nil)
		return
	}

	// 9. 트랜잭션 커밋
	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR] google verify: commit transaction: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to commit transaction", nil)
		return
	}

	// 10. 응답
	// 토큰이 담긴 응답은 어디에도 캐시되지 않게 합니다 (RFC 6749 5.1)
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusOK, GoogleVerifyResponse{
		TokenPairResponse: *pair,
		Token:             pair.AccessToken,
		User:              user,
	})
}
