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
// id_token만 필수입니다. email·name·picture는 선택이며, 서버는 ID Token의 검증된 값을 우선 사용합니다
type GoogleVerifyRequest struct {
	// IDToken은 Google이 발급한 ID Token입니다 (필수)
	IDToken string `json:"id_token"`
	// Email은 선택입니다. 저장되는 이메일은 항상 ID Token의 검증된 이메일이며 이 값은 쓰지 않습니다
	Email string `json:"email"`
	// Name은 선택입니다. ID Token에 이름이 없을 때만 사용합니다
	Name string `json:"name"`
	// Picture는 선택입니다. ID Token에 사진이 없을 때만 사용합니다
	Picture string `json:"picture"`
}

// GoogleVerifyResponse는 Google ID Token 검증 성공 응답입니다
type GoogleVerifyResponse struct {
	TokenPairResponse

	User *models.User `json:"user"`
}

// GoogleVerify는 Google ID Token을 검증하고 Access Token과 Refresh Token을 발급합니다
//
// @Summary      Google OAuth 인증 및 토큰 발급
// @Description  Google ID Token을 검증하고 사용자를 생성/조회한 후 Access Token과 Refresh Token을 발급합니다
// @Description  id_token만 필수입니다. email·name·picture는 선택이며, 서버는 ID Token의 검증된 값을 우선 사용합니다
// @Description  이메일이 검증되지 않은(email_verified가 true가 아닌) ID Token은 401 INVALID_ID_TOKEN입니다
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body GoogleVerifyRequest true "Google 인증 정보"
// @Success      200 {object} GoogleVerifyResponse "토큰 쌍 및 사용자 정보"
// @Header       200 {string} Cache-Control "no-store"
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
		user = newUserFromGoogle(payload, req)

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
		User:              user,
	})
}

// newUserFromGoogle은 신규 사용자 정보를 만듭니다
// 이메일은 Google이 검증한 ID Token 값만 씁니다. 요청 본문은 클라이언트가 임의로 바꿀 수 있기 때문입니다
// 이름과 사진은 ID Token 값을 우선 쓰고, 없으면 요청 본문 값을 씁니다
// 이름이 둘 다 없으면 users.name이 필수이므로 이메일을 이름으로 씁니다
func newUserFromGoogle(payload *auth.GoogleIDTokenPayload, req GoogleVerifyRequest) *models.User {
	name := firstNonEmpty(payload.Name, req.Name, payload.Email)
	picture := firstNonEmpty(payload.Picture, req.Picture)

	return &models.User{
		Email:      payload.Email,
		Name:       name,
		PictureURL: picture,
		GoogleID:   payload.GoogleID,
	}
}

// firstNonEmpty는 인자 중 처음으로 비어 있지 않은 문자열을 반환합니다 (모두 비면 빈 문자열)
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
