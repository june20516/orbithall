package ratelimit

import (
	"context"
	"sync"
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

// setLastSeen은 테스트에서 특정 키의 lastSeen을 원하는 시각으로 강제 설정합니다
func setLastSeen(t *testing.T, rl *RateLimiter, key string, at time.Time) {
	t.Helper()
	v, exists := rl.visitors.Load(key)
	if !exists {
		t.Fatalf("key %q not found", key)
	}
	v.(*entry).lastSeen.Store(at.UnixNano())
}

// TestCleanupIdle_RemovesOnlyStaleEntries는 idle 시간보다 오래 쓰이지 않은 항목만 지워지는지 테스트합니다
func TestCleanupIdle_RemovesOnlyStaleEntries(t *testing.T) {
	// Given: 두 개의 키가 등록된 RateLimiter
	rl := NewRateLimiter(rate.Limit(10), 5)
	rl.GetLimiter("stale-key")
	rl.GetLimiter("fresh-key")

	// When: stale-key는 40분 전, fresh-key는 5분 전에 사용된 것처럼 lastSeen을 조작한 뒤
	// idle 30분 기준으로 CleanupIdle 호출
	now := time.Now()
	setLastSeen(t, rl, "fresh-key", now.Add(-5*time.Minute))
	setLastSeen(t, rl, "stale-key", now.Add(-40*time.Minute))

	removed := rl.CleanupIdle(now, 30*time.Minute)

	// Then: stale-key만 지워지고 fresh-key는 남아야 함
	if removed != 1 {
		t.Fatalf("Expected 1 entry removed, got %d", removed)
	}
	if _, exists := rl.visitors.Load("stale-key"); exists {
		t.Fatal("Expected stale-key to be removed")
	}
	if _, exists := rl.visitors.Load("fresh-key"); !exists {
		t.Fatal("Expected fresh-key to remain")
	}
}

// TestGetLimiter_UpdatesLastSeen는 GetLimiter 호출마다 lastSeen이 갱신되는지 테스트합니다
func TestGetLimiter_UpdatesLastSeen(t *testing.T) {
	// Given: 하나의 키가 등록된 RateLimiter
	rl := NewRateLimiter(rate.Limit(10), 5)
	rl.GetLimiter("key")

	v, _ := rl.visitors.Load("key")
	e := v.(*entry)
	e.lastSeen.Store(time.Now().Add(-1 * time.Hour).UnixNano())
	oldLastSeen := e.lastSeen.Load()

	// When: 같은 키로 GetLimiter를 다시 호출
	rl.GetLimiter("key")

	// Then: lastSeen이 갱신되어 있어야 함
	if e.lastSeen.Load() <= oldLastSeen {
		t.Fatal("Expected lastSeen to be updated after GetLimiter call")
	}
}

// TestStartCleanup_StopsOnContextCancel은 ctx가 취소되면 정리 고루틴이 종료되는지 테스트합니다
func TestStartCleanup_StopsOnContextCancel(t *testing.T) {
	// Given: 짧은 주기로 StartCleanup을 시작한 RateLimiter
	rl := NewRateLimiter(rate.Limit(10), 5)
	rl.GetLimiter("key")

	ctx, cancel := context.WithCancel(context.Background())
	rl.StartCleanup(ctx, 10*time.Millisecond, 1*time.Millisecond)

	// When: 정리가 최소 한 번 동작할 시간을 준 뒤 ctx를 취소
	// (idle을 극히 짧게 두어 다음 tick에서 key가 지워지는지로 동작 여부를 확인)
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		if _, exists := rl.visitors.Load("key"); !exists {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Expected StartCleanup goroutine to remove idle entry before deadline")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()

	// Then: cancel 이후 고루틴이 종료되어 더 이상 정리가 일어나지 않아야 함 (goroutine 누수 없음)
	// 새 키를 등록하고 한동안 기다려도 삭제되지 않으면 고루틴이 멈춘 것으로 판단
	time.Sleep(20 * time.Millisecond)
	rl.GetLimiter("after-cancel")
	time.Sleep(50 * time.Millisecond)
	if _, exists := rl.visitors.Load("after-cancel"); !exists {
		t.Fatal("Expected entry to remain after ctx cancel, but cleanup goroutine still running")
	}
}

// TestConcurrentGetLimiterAndCleanupIdle은 GetLimiter와 CleanupIdle을 동시에 호출해도
// race detector가 경합을 검출하지 않는지 확인합니다
func TestConcurrentGetLimiterAndCleanupIdle(t *testing.T) {
	// Given: RateLimiter
	rl := NewRateLimiter(rate.Limit(1000), 1000)

	var wg sync.WaitGroup

	// When: 여러 고루틴이 동시에 GetLimiter를 호출하고
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				rl.GetLimiter("concurrent-key")
			}
		}(i)
	}

	// 동시에 다른 고루틴이 CleanupIdle을 반복 호출
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				rl.CleanupIdle(time.Now(), time.Millisecond)
			}
		}
	}()

	wg.Wait()
	close(stop)

	// Then: race detector가 아무것도 검출하지 않으면 성공 (go test -race로 확인)
}
