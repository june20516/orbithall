package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"golang.org/x/time/rate"
)

// TestRateLimitMiddleware_AllowWithinLimit는 제한 이내의 요청이 성공하는지 테스트합니다
func TestRateLimitMiddleware_AllowWithinLimit(t *testing.T) {
	// Given: 10 req/sec, burst 5인 미들웨어
	rl := NewRateLimiter(rate.Limit(10), 5)
	middleware := RateLimitMiddleware(rl)

	// 테스트용 핸들러 (200 OK 반환)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// When: burst 이내의 요청을 보냄 (5번)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Then: 모든 요청이 200 OK를 받아야 함
		if rec.Code != http.StatusOK {
			t.Fatalf("Request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}
}

// TestRateLimitMiddleware_Deny429OnExceed는 제한 초과 시 429 응답을 반환하는지 테스트합니다
func TestRateLimitMiddleware_Deny429OnExceed(t *testing.T) {
	// Given: 1 req/sec, burst 2인 미들웨어 (테스트를 위해 낮은 값)
	rl := NewRateLimiter(rate.Limit(1), 2)
	middleware := RateLimitMiddleware(rl)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// When: burst 이내의 요청 (2번) - 성공해야 함
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}

	// When: burst 초과 요청 (3번째)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Then: 429 Too Many Requests를 반환해야 함
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("Expected status 429, got %d", rec.Code)
	}
}

// TestRateLimitMiddleware_RetryAfterHeader는 429 응답 시 Retry-After 헤더가 포함되는지 테스트합니다
func TestRateLimitMiddleware_RetryAfterHeader(t *testing.T) {
	// Given: 1 req/sec, burst 1인 미들웨어
	rl := NewRateLimiter(rate.Limit(1), 1)
	middleware := RateLimitMiddleware(rl)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// When: 첫 번째 요청 (성공)
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// When: 두 번째 요청 (제한 초과)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Then: Retry-After 헤더가 존재해야 함
	retryAfter := rec2.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("Expected Retry-After header, but got empty")
	}

	// Then: Retry-After 값이 숫자여야 함 (초 단위)
	if _, err := strconv.Atoi(retryAfter); err != nil {
		t.Fatalf("Expected Retry-After to be a number, got %s", retryAfter)
	}
}

// TestRateLimitMiddleware_IndependentIPs는 다른 IP가 독립적으로 제한되는지 테스트합니다
func TestRateLimitMiddleware_IndependentIPs(t *testing.T) {
	// Given: 1 req/sec, burst 1인 미들웨어
	rl := NewRateLimiter(rate.Limit(1), 1)
	middleware := RateLimitMiddleware(rl)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// When: IP1에서 요청 (성공)
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("IP1 first request: expected 200, got %d", rec1.Code)
	}

	// When: IP1에서 두 번째 요청 (제한 초과 - 429)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("IP1 second request: expected 429, got %d", rec2.Code)
	}

	// When: IP2에서 요청 (IP1의 제한과 무관하게 성공해야 함)
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "192.168.1.2:12345"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)

	// Then: IP2는 독립적인 Limiter를 가지므로 성공해야 함
	if rec3.Code != http.StatusOK {
		t.Fatalf("IP2 first request: expected 200, got %d", rec3.Code)
	}
}

// TestRateLimitMiddleware_XForwardedFor는 X-Forwarded-For 헤더를 사용하는지 테스트합니다
func TestRateLimitMiddleware_XForwardedFor(t *testing.T) {
	// Given: 1 req/sec, burst 1인 미들웨어
	rl := NewRateLimiter(rate.Limit(1), 1)
	middleware := RateLimitMiddleware(rl)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// When: X-Forwarded-For 헤더를 포함한 요청 (성공)
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req1.RemoteAddr = "192.168.1.1:12345" // 이 IP는 무시되어야 함
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("First request: expected 200, got %d", rec1.Code)
	}

	// When: 같은 X-Forwarded-For IP로 두 번째 요청 (제한 초과)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req2.RemoteAddr = "192.168.1.99:54321" // RemoteAddr가 달라도 같은 IP로 판단
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Then: X-Forwarded-For의 첫 번째 IP(10.0.0.1)를 기준으로 제한되어야 함
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("Second request: expected 429, got %d", rec2.Code)
	}
}
