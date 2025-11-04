package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
)

// AuthHandler는 인증 관련 HTTP 요청을 처리합니다
type AuthHandler struct {
	db *sql.DB
}

// NewAuthHandler는 AuthHandler의 새 인스턴스를 생성합니다
func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{
		db: db,
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
	Token string        `json:"token"`
	User  *models.User `json:"user"`
}

// GoogleVerify는 Google ID Token을 검증하고 백엔드 JWT를 발급합니다
//
// @Summary      Google OAuth 인증 및 JWT 발급
// @Description  Google ID Token을 검증하고 사용자를 생성/조회한 후 백엔드 JWT를 발급합니다
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body GoogleVerifyRequest true "Google 인증 정보"
// @Success      200 {object} GoogleVerifyResponse "JWT 토큰 및 사용자 정보"
// @Failure      400 {string} string "Invalid request body or missing required fields"
// @Failure      401 {string} string "Invalid Google ID Token"
// @Failure      500 {string} string "Internal server error"
// @Router       /auth/google/verify [post]
func (h *AuthHandler) GoogleVerify(w http.ResponseWriter, r *http.Request) {
	// 1. Content-Type 검증
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// 2. JSON 요청 파싱
	var req GoogleVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 3. 입력 검증
	if req.IDToken == "" {
		http.Error(w, "id_token is required", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// 4. Google ID Token 검증
	payload, err := auth.VerifyGoogleIDToken(r.Context(), req.IDToken)
	if err != nil {
		if err == auth.ErrInvalidIDToken {
			http.Error(w, "Invalid Google ID Token", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to verify Google ID Token", http.StatusInternalServerError)
		return
	}

	// 5. 트랜잭션 시작
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// 6. Google ID로 사용자 조회
	user, err := database.GetUserByGoogleID(r.Context(), tx, payload.GoogleID)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
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
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}

	// 8. 트랜잭션 커밋
	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// 9. JWT 생성
	token, err := auth.GenerateJWT(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate JWT", http.StatusInternalServerError)
		return
	}

	// 10. 응답
	response := GoogleVerifyResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
