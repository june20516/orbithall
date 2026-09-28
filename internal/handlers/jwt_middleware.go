package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
)

// userContextKey는 Context에 사용자 정보를 저장할 때 사용하는 키입니다
const userContextKey contextKey = "user"

// JWTAuthMiddleware는 JWT 기반 인증 미들웨어입니다
// Authorization 헤더에서 Bearer 토큰을 추출하고 검증합니다
// 실패 시 {"error":{"code","message"}} 형식으로 응답합니다
func JWTAuthMiddleware(db database.DBTX) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Authorization 헤더 추출
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, http.StatusUnauthorized, ErrMissingToken, "Authorization header is required", nil)
				return
			}

			// 2. Bearer 형식 검증
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondError(w, http.StatusUnauthorized, ErrInvalidToken, "Authorization header must be in format: Bearer {token}", nil)
				return
			}

			tokenString := parts[1]

			// 3. JWT 토큰 검증
			claims, err := auth.ValidateJWT(tokenString)
			if err != nil {
				if err == auth.ErrExpiredToken {
					respondError(w, http.StatusUnauthorized, ErrExpiredToken, "Token has expired", nil)
					return
				}
				respondError(w, http.StatusUnauthorized, ErrInvalidToken, "Invalid token", nil)
				return
			}

			// 4. 사용자 조회
			user, err := database.GetUserByID(r.Context(), db, claims.UserID)
			if err != nil {
				// details는 응답 본문에 들어가므로 원인은 서버 로그에만 남깁니다
				log.Printf("[ERROR] jwt auth: get user: %v", err)
				respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get user", nil)
				return
			}

			// 5. 사용자 존재 여부 확인
			if user == nil {
				respondError(w, http.StatusUnauthorized, ErrUserNotFound, "User not found", nil)
				return
			}

			// 6. Context에 사용자 정보 저장
			ctx := SetUserInContext(r.Context(), user)

			// 7. 다음 핸들러 호출
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SetUserInContext는 Context에 사용자 정보를 저장합니다
func SetUserInContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUserFromContext는 Context에서 사용자 정보를 추출합니다
// 사용자 정보가 없으면 nil을 반환합니다
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(userContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
