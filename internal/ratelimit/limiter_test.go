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

// waitUntil은 condition이 true가 될 때까지 짧은 간격으로 폴링하고, timeout 안에
// 만족되지 않으면 msg로 테스트를 실패시킵니다. 고정된 sleep 대신 조건 자체를 기다려
// 부하가 걸린 환경(CI, -race)에서도 결정적으로 동작합니다
func waitUntil(t *testing.T, timeout time.Duration, condition func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if condition() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(5 * time.Millisecond)
	}
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
	// Given: 짧은 주기로 정리 고루틴을 시작한 RateLimiter
	// (패키지 내부의 startCleanupLoop를 직접 호출해 종료 시 닫히는 done 채널을 받습니다.
	// StartCleanup은 이 함수를 감싸기만 하므로 동작은 동일합니다)
	rl := NewRateLimiter(rate.Limit(10), 5)
	rl.GetLimiter("key")

	ctx, cancel := context.WithCancel(context.Background())
	done := rl.startCleanupLoop(ctx, 10*time.Millisecond, 1*time.Millisecond)

	// When: 정리가 최소 한 번 동작할 때까지 폴링으로 기다린 뒤 ctx를 취소
	// (idle을 극히 짧게 두어 다음 tick에서 key가 지워지는지로 동작 여부를 확인)
	waitUntil(t, 500*time.Millisecond, func() bool {
		_, exists := rl.visitors.Load("key")
		return !exists
	}, "Expected StartCleanup goroutine to remove idle entry before deadline")

	cancel()

	// 고루틴이 실제로 종료될 때까지 done 채널을 기다립니다 (sleep으로 추측하지 않음)
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Expected cleanup goroutine to exit after ctx cancel")
	}

	// Then: 고루틴이 이미 종료되었으므로, 새 키를 등록해도 더 이상 정리가 일어나지 않아야 함
	rl.GetLimiter("after-cancel")
	if _, exists := rl.visitors.Load("after-cancel"); !exists {
		t.Fatal("Expected entry to remain after ctx cancel, but cleanup goroutine still running")
	}
}

// TestConcurrentGetLimiterAndCleanupIdle은 GetLimiter와 CleanupIdle을 동시에 호출해도
// race detector가 경합을 검출하지 않는지 확인합니다
func TestConcurrentGetLimiterAndCleanupIdle(t *testing.T) {
	// Given: RateLimiter
	rl := NewRateLimiter(rate.Limit(1000), 1000)

	var workers sync.WaitGroup

	// When: 여러 고루틴이 동시에 GetLimiter를 호출하고
	for i := 0; i < 20; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 50; j++ {
				rl.GetLimiter("concurrent-key")
			}
		}()
	}

	// 동시에 다른 고루틴이 CleanupIdle을 반복 호출
	stop := make(chan struct{})
	var cleanup sync.WaitGroup
	cleanup.Add(1)
	go func() {
		defer cleanup.Done()
		for {
			select {
			case <-stop:
				return
			default:
				rl.CleanupIdle(time.Now(), time.Millisecond)
			}
		}
	}()

	workers.Wait()
	close(stop)
	cleanup.Wait() // 테스트에서 띄운 정리 고루틴이 실제로 종료될 때까지 대기

	// Then: race detector가 아무것도 검출하지 않으면 성공 (go test -race로 확인)
}
