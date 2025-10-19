package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
)

// ============================================
// 에러 코드 상수
// ============================================

const (
	// API 키 관련 에러
	ErrMissingAPIKey = "MISSING_API_KEY" // API 키 헤더 없음
	ErrInvalidAPIKey = "INVALID_API_KEY" // API 키 형식 오류 또는 존재하지 않음
	ErrSiteInactive  = "SITE_INACTIVE"   // 비활성화된 사이트
	ErrInvalidOrigin = "INVALID_ORIGIN"  // CORS Origin 불일치

	// 입력 검증 에러
	ErrInvalidInput = "INVALID_INPUT" // 입력 검증 실패

	// 리소스 관련 에러
	ErrPostNotFound    = "POST_NOT_FOUND"    // 포스트 없음
	ErrCommentNotFound = "COMMENT_NOT_FOUND" // 댓글 없음

	// 권한 관련 에러
	ErrWrongPassword   = "WRONG_PASSWORD"    // 비밀번호 불일치
	ErrEditTimeExpired = "EDIT_TIME_EXPIRED" // 수정 가능 시간 초과

	// Rate limiting 에러
	ErrRateLimitExceeded = "RATE_LIMIT_EXCEEDED" // Rate limit 초과

	// 서버 에러
	ErrInternalServer = "INTERNAL_SERVER_ERROR" // 서버 내부 오류
)

// ============================================
// Context Key 정의
// ============================================

// contextKey는 Context에 저장되는 값의 키 타입입니다
// 비공개 타입으로 정의하여 다른 패키지와의 충돌을 방지합니다
type contextKey string

// siteContextKey는 Context에 사이트 정보를 저장할 때 사용하는 키입니다
const siteContextKey contextKey = "site"

// ============================================
// 응답 구조체
// ============================================

// ErrorResponse는 API 에러 응답의 표준 구조입니다
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail은 에러의 상세 정보를 담습니다
type ErrorDetail struct {
	Code    string      `json:"code"`              // 에러 코드 (예: MISSING_API_KEY)
	Message string      `json:"message"`           // 사람이 읽을 수 있는 에러 메시지
	Details interface{} `json:"details,omitempty"` // 선택적 추가 정보 (예: 입력 검증 오류 목록)
}

// ============================================
// 헬퍼 함수
// ============================================

// respondJSON은 JSON 응답을 클라이언트에 전송합니다
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// respondError는 에러 응답을 클라이언트에 전송합니다
func respondError(w http.ResponseWriter, statusCode int, code string, message string, details interface{}) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	respondJSON(w, statusCode, response)
}

// GetSiteFromContext는 Context에서 사이트 정보를 추출합니다
// 미들웨어에서 저장한 사이트 정보를 핸들러에서 사용할 때 호출합니다
func GetSiteFromContext(ctx context.Context) *models.Site {
	site, ok := ctx.Value(siteContextKey).(*models.Site)
	if !ok {
		return nil
	}
	return site
}

// isOriginAllowed는 요청 Origin이 허용된 Origin 목록에 포함되는지 확인합니다
// 대소문자를 구분하지 않고 비교합니다
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		// 대소문자 무시하고 비교
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

// ============================================
// 미들웨어
// ============================================

// AuthMiddleware는 API 키 기반 인증을 수행하는 미들웨어입니다
// 모든 API 엔드포인트에 적용되어야 합니다
//
// 처리 흐름:
// 1. X-Orbithall-API-Key 헤더 추출
// 2. API 키로 사이트 조회 (캐시 사용)
// 3. 사이트 활성화 확인
// 4. CORS Origin 검증
// 5. Context에 사이트 정보 저장
func AuthMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. API 키 추출
			apiKey := r.Header.Get("X-Orbithall-API-Key")
			if apiKey == "" {
				respondError(w, http.StatusUnauthorized, ErrMissingAPIKey, "API key is required", nil)
				return
			}

			// 2. API 키로 사이트 조회 (캐시 자동 사용)
			site, err := database.GetSiteByAPIKey(db, apiKey)
			if err != nil {
				// GetSiteByAPIKey는 is_active=true인 사이트만 조회하므로
				// 에러가 발생하면 API 키가 잘못되었거나 사이트가 비활성화된 것
				respondError(w, http.StatusForbidden, ErrInvalidAPIKey, "Invalid API key", nil)
				return
			}

			// 3. 사이트 활성화 확인 (이미 GetSiteByAPIKey에서 처리되지만 명시적 확인)
			if !site.IsActive {
				respondError(w, http.StatusForbidden, ErrSiteInactive, "Site is inactive", nil)
				return
			}

			// 4. CORS Origin 검증
			// Origin 헤더가 있는 경우에만 검증 (브라우저 요청만 Origin 헤더 포함)
			origin := r.Header.Get("Origin")
			if origin != "" {
				if !isOriginAllowed(origin, site.CORSOrigins) {
					respondError(w, http.StatusForbidden, ErrInvalidOrigin, "Origin not allowed", nil)
					return
				}
			}

			// 5. Context에 사이트 정보 저장
			ctx := context.WithValue(r.Context(), siteContextKey, site)

			// 6. 다음 핸들러 호출
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
