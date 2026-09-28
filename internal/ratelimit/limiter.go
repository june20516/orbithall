package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// entry는 하나의 키에 대한 *rate.Limiter와 마지막 사용 시각을 함께 보관합니다
// lastSeen은 GetLimiter가 호출될 때마다 원자적으로 갱신되며, CleanupIdle이 오래
// 쓰이지 않은 키를 판별하는 기준이 됩니다 (UnixNano)
type entry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64
}

// RateLimiter는 IP별 요청 제한을 관리합니다
// sync.Map을 사용하여 thread-safe하게 IP별 Limiter를 저장합니다
type RateLimiter struct {
	// visitors는 IP 주소(또는 로그인 계열 ID)를 키로, *entry를 값으로 저장합니다
	visitors sync.Map

	// limit는 초당 허용되는 요청 수입니다 (토큰 생성 속도)
	limit rate.Limit

	// burst는 한 번에 허용되는 최대 요청 수입니다 (버킷 크기)
	burst int
}

// NewRateLimiter는 새로운 RateLimiter를 생성합니다
//
// 파라미터:
//   - limit: 초당 허용되는 요청 수 (예: 10 = 10 req/sec)
//   - burst: 한 번에 허용되는 최대 요청 수 (예: 5 = 5개까지 연속 요청 가능)
//
// 예시:
//   - NewRateLimiter(10, 5): 초당 10개, 최대 5개까지 burst 허용
func NewRateLimiter(limit rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		visitors: sync.Map{},
		limit:    limit,
		burst:    burst,
	}
}

// GetLimiter는 키(IP 또는 로그인 계열 ID)에 대한 Limiter를 반환합니다
// 키가 처음 요청되면 새 Limiter를 생성하고, 이미 존재하면 기존 Limiter를 반환합니다
// 호출될 때마다 해당 키의 lastSeen을 현재 시각으로 갱신하여, CleanupIdle이
// 실제로 사용 중인 키를 지우지 않도록 합니다
//
// 토큰 버킷 알고리즘:
//   - burst만큼의 토큰으로 시작
//   - 요청마다 토큰 1개 소비
//   - limit 속도로 토큰 재충전
//   - 토큰이 없으면 요청 거부
func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
	now := time.Now().UnixNano()

	// 키에 해당하는 entry 조회
	if v, exists := rl.visitors.Load(key); exists {
		e := v.(*entry)
		e.lastSeen.Store(now)
		return e.limiter
	}

	// 새 entry 생성
	e := &entry{limiter: rate.NewLimiter(rl.limit, rl.burst)}
	e.lastSeen.Store(now)

	// sync.Map에 저장 (thread-safe)
	// LoadOrStore는 다른 고루틴이 동시에 같은 키로 요청했을 때를 대비합니다
	actual, loaded := rl.visitors.LoadOrStore(key, e)
	actualEntry := actual.(*entry)
	if loaded {
		// 이미 다른 고루틴이 먼저 저장했다면, 그 entry를 최신 사용 시각으로 갱신하고 사용합니다
		actualEntry.lastSeen.Store(now)
	}

	return actualEntry.limiter
}

// CleanupIdle은 now 기준으로 idle보다 오래 사용되지 않은 항목을 삭제하고, 지운 항목 수를 반환합니다
// now를 인자로 받아 테스트에서 시각을 주입할 수 있게 합니다
//
// idle은 토큰 버킷이 가득 찰 만큼 충분히 길어야 삭제가 제한을 느슨하게 만들지 않습니다:
// 항목을 지운 뒤 같은 키로 다시 요청이 오면 burst만큼 가득 찬 새 Limiter가 생성되는데,
// 이는 idle 시간(≥ burst / limit) 동안 토큰이 어차피 가득 재충전되었을 상태와 같습니다.
// 즉 idle < burst / limit 이면, 아직 토큰이 다 차지 않은 항목을 지우고 새로 만들어
// 실질적으로 버킷을 리필해주는 셈이 되어 제한이 우회됩니다.
func (rl *RateLimiter) CleanupIdle(now time.Time, idle time.Duration) int {
	cutoff := now.Add(-idle).UnixNano()
	removed := 0

	rl.visitors.Range(func(key, value any) bool {
		e := value.(*entry)
		if e.lastSeen.Load() < cutoff {
			rl.visitors.Delete(key)
			removed++
		}
		return true
	})

	return removed
}

// StartCleanup은 interval마다 CleanupIdle(time.Now(), idle)을 호출하는 고루틴을 시작합니다
// ctx가 취소되면 고루틴이 종료됩니다 (goroutine 누수 방지)
func (rl *RateLimiter) StartCleanup(ctx context.Context, interval, idle time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rl.CleanupIdle(time.Now(), idle)
			}
		}
	}()
}
