package ratelimit

import (
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter는 IP별 요청 제한을 관리합니다
// sync.Map을 사용하여 thread-safe하게 IP별 Limiter를 저장합니다
type RateLimiter struct {
	// visitors는 IP 주소를 키로, *rate.Limiter를 값으로 저장합니다
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

// GetLimiter는 IP 주소에 대한 Limiter를 반환합니다
// IP가 처음 요청되면 새 Limiter를 생성하고, 이미 존재하면 기존 Limiter를 반환합니다
//
// 토큰 버킷 알고리즘:
//   - burst만큼의 토큰으로 시작
//   - 요청마다 토큰 1개 소비
//   - limit 속도로 토큰 재충전
//   - 토큰이 없으면 요청 거부
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	// IP에 해당하는 Limiter 조회
	if limiter, exists := rl.visitors.Load(ip); exists {
		// 이미 존재하면 기존 Limiter 반환
		return limiter.(*rate.Limiter)
	}

	// 새 Limiter 생성
	limiter := rate.NewLimiter(rl.limit, rl.burst)

	// sync.Map에 저장 (thread-safe)
	// LoadOrStore는 다른 고루틴이 동시에 같은 IP로 요청했을 때를 대비합니다
	actual, _ := rl.visitors.LoadOrStore(ip, limiter)

	return actual.(*rate.Limiter)
}
