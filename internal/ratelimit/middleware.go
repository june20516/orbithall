package ratelimit

import (
	"net/http"

	"github.com/june20516/orbithall/internal/httputil"
)

// RateLimitMiddleware는 IP 기반 요청 제한 미들웨어를 반환합니다
// 제한을 초과하면 429 Too Many Requests 응답을 반환합니다
//
// 사용 예시:
//
//	rl := NewRateLimiter(rate.Limit(10), 5)
//	r.Use(RateLimitMiddleware(rl))
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// IP 주소 추출 (프록시 환경 고려)
			ip := httputil.GetIPAddress(r)

			// IP별 Limiter 가져오기
			limiter := rl.GetLimiter(ip)

			// 요청 허용 여부 확인
			if !limiter.Allow() {
				// 제한 초과 - 429 응답
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Too many requests. Please try again later."}`))
				return
			}

			// 제한 이내 - 다음 핸들러 호출
			next.ServeHTTP(w, r)
		})
	}
}
