# [WIP] Rate Limiting 구현

## 작성일
2025-10-14

## 우선순위
- [ ] 긴급
- [x] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
API 남용 방지를 위한 요청 제한(Rate Limiting) 기능 구현

## 작업 목적
스팸 댓글, DDoS 공격, API 남용을 방지하여 서비스 안정성 확보

## 작업 범위

### 포함 사항
- IP 기반 Rate Limiting
- API 키 기반 Rate Limiting (사이트별 제한)
- 429 Too Many Requests 응답
- Retry-After 헤더
- 메모리 기반 토큰 버킷 알고리즘

### 제외 사항
- Redis 기반 분산 Rate Limiting (추후 필요시)
- 사용자별 제한 (현재는 익명 댓글만)
- 동적 제한 조정 (관리자 기능)

## 기술적 접근

### 사용할 기술/라이브러리
- `golang.org/x/time/rate` (토큰 버킷)
- 또는 `github.com/go-chi/httprate` (Chi 전용)
- sync.Map (thread-safe 캐싱)

### Rate Limit 정책 (초안)
```
IP 기반:
- 댓글 작성: 10회/시간
- 댓글 조회: 100회/분
- 댓글 수정/삭제: 5회/시간

API 키(사이트) 기반:
- 전체 요청: 1000회/시간
```

## 구현 단계

### 1. Rate Limiter 패키지 생성
- `internal/ratelimit/limiter.go`
- IP별, API 키별 제한 로직
- 토큰 버킷 알고리즘

### 2. 미들웨어 구현
- IP Rate Limit 미들웨어
- API Key Rate Limit 미들웨어
- 429 응답 및 헤더 설정

### 3. 라우터에 적용
- 엔드포인트별 제한 설정
- 우선순위: 작성 > 수정/삭제 > 조회

### 4. 테스트
- 제한 초과 시 429 반환 확인
- 시간 경과 후 복구 확인
- 동시 요청 처리 확인

## 검증 방법

### 테스트 시나리오
1. **IP Rate Limit**
   - 동일 IP에서 연속 요청
   - 제한 초과 시 429 확인
   - Retry-After 헤더 확인

2. **API Key Rate Limit**
   - 동일 API 키로 대량 요청
   - 사이트별 격리 확인

3. **복구 테스트**
   - 제한 후 시간 경과
   - 정상 요청 가능 확인

## 의존성
- 선행 작업: 댓글 CRUD API 구현
- 후속 작업: Railway 배포

## 예상 소요 시간
- 예상: 2-3시간

## 주의사항
- 프록시 환경에서 실제 IP 추출 (X-Forwarded-For, X-Real-IP)
- 메모리 누수 방지 (오래된 항목 정리)
- Railway 환경 고려 (단일 인스턴스 가정)

---

## 작업 이력

### [2025-10-14] 작업 문서 초안 작성
- 러프한 구조 작성
- 구체적인 정책은 작업 시작 시 확정
