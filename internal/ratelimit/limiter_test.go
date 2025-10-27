package ratelimit

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// TestGetLimiter_NewIP는 새로운 IP에 대해 새 Limiter를 생성하는지 테스트합니다
func TestGetLimiter_NewIP(t *testing.T) {
	// Given: 비어있는 RateLimiter
	rl := NewRateLimiter(rate.Limit(10), 5)

	// When: 새 IP로 Limiter를 요청
	ip := "192.168.1.1"
	limiter := rl.GetLimiter(ip)

	// Then: Limiter가 생성되어야 함
	if limiter == nil {
		t.Fatal("Expected limiter to be created, but got nil")
	}
}

// TestGetLimiter_ExistingIP는 기존 IP에 대해 동일한 Limiter를 반환하는지 테스트합니다
func TestGetLimiter_ExistingIP(t *testing.T) {
	// Given: RateLimiter에 이미 등록된 IP
	rl := NewRateLimiter(rate.Limit(10), 5)
	ip := "192.168.1.1"
	limiter1 := rl.GetLimiter(ip)

	// When: 같은 IP로 다시 Limiter를 요청
	limiter2 := rl.GetLimiter(ip)

	// Then: 동일한 Limiter 인스턴스를 반환해야 함
	if limiter1 != limiter2 {
		t.Fatal("Expected same limiter instance for same IP, but got different instances")
	}
}

// TestAllow_WithinLimit는 제한 이내의 요청이 허용되는지 테스트합니다
func TestAllow_WithinLimit(t *testing.T) {
	// Given: 10 req/sec, burst 5인 RateLimiter
	rl := NewRateLimiter(rate.Limit(10), 5)
	ip := "192.168.1.1"
	limiter := rl.GetLimiter(ip)

	// When: burst 이내의 요청을 보냄 (5번)
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Fatalf("Request %d should be allowed within burst limit", i+1)
		}
	}

	// Then: 모든 요청이 허용되어야 함 (위 반복문에서 검증)
}

// TestAllow_ExceedLimit는 제한을 초과하는 요청이 거부되는지 테스트합니다
func TestAllow_ExceedLimit(t *testing.T) {
	// Given: 1 req/sec, burst 2인 RateLimiter (테스트를 위해 낮은 값 설정)
	rl := NewRateLimiter(rate.Limit(1), 2)
	ip := "192.168.1.1"
	limiter := rl.GetLimiter(ip)

	// When: burst 이내의 요청 (2번) - 허용되어야 함
	for i := 0; i < 2; i++ {
		if !limiter.Allow() {
			t.Fatalf("Request %d should be allowed within burst limit", i+1)
		}
	}

	// When: burst를 초과하는 요청 (3번째) - 즉시 실행하면 거부되어야 함
	if limiter.Allow() {
		t.Fatal("Request should be denied when exceeding burst limit")
	}

	// Then: 시간이 경과하면 (1초) 다시 허용되어야 함
	time.Sleep(1 * time.Second)
	if !limiter.Allow() {
		t.Fatal("Request should be allowed after rate limit period")
	}
}

// TestGetLimiter_DifferentIPs는 서로 다른 IP가 독립적인 Limiter를 가지는지 테스트합니다
func TestGetLimiter_DifferentIPs(t *testing.T) {
	// Given: RateLimiter
	rl := NewRateLimiter(rate.Limit(10), 5)

	// When: 서로 다른 IP로 Limiter를 요청
	limiter1 := rl.GetLimiter("192.168.1.1")
	limiter2 := rl.GetLimiter("192.168.1.2")

	// Then: 서로 다른 Limiter 인스턴스를 반환해야 함
	if limiter1 == limiter2 {
		t.Fatal("Expected different limiter instances for different IPs")
	}
}
