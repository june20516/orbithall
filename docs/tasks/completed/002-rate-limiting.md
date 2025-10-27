# Rate Limiting 구현

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
- 선행 작업: ✅ 댓글 CRUD API 완료 (001-comment-crud-api)
- 선행 작업: ✅ 프로덕션 배포 완료 (003-deployment)
- 후속 작업: 없음

## 예상 소요 시간
- 예상: 2-3시간

## 주의사항
- 프록시 환경에서 실제 IP 추출 (X-Forwarded-For, X-Real-IP)
- 메모리 누수 방지 (오래된 항목 정리)
- Render 환경 고려 (단일 인스턴스 가정)

## 구현 체크리스트

### 1. 준비 단계
- [x] `golang.org/x/time/rate` 의존성 추가
- [x] `internal/ratelimit` 디렉토리 생성

### 2. Rate Limiter 테스트 작성 (Red)
- [x] `internal/ratelimit/limiter_test.go` 작성
  - [x] GetLimiter 테스트 (새 IP는 새 Limiter 생성)
  - [x] GetLimiter 테스트 (기존 IP는 기존 Limiter 반환)
  - [x] Allow 테스트 (제한 이내 요청)
  - [x] Allow 테스트 (제한 초과 요청)
  - [x] 다른 IP는 독립적인 Limiter 테스트 추가
  - [x] 테스트 실패 확인 (build failed)

### 3. Rate Limiter 구현 (Green)
- [x] `internal/ratelimit/limiter.go` 생성
  - [x] IP별 Limiter 저장 구조 (sync.Map)
  - [x] 토큰 버킷 생성 함수 (NewRateLimiter)
  - [x] GetLimiter 함수 (LoadOrStore로 thread-safe)
  - [x] 모든 테스트 통과 확인 (5개 PASS)

### 4. 미들웨어 테스트 작성 (Red)
- [x] `internal/ratelimit/middleware_test.go` 작성
  - [x] 제한 이내 요청 성공 테스트
  - [x] 제한 초과 시 429 응답 테스트
  - [x] Retry-After 헤더 확인 테스트
  - [x] 다른 IP는 독립적으로 제한 테스트
  - [x] X-Forwarded-For 헤더 처리 테스트 추가
  - [x] 테스트 실패 확인 (build failed)

### 5. 미들웨어 구현 (Green)
- [x] `internal/ratelimit/middleware.go` 생성
  - [x] IP Rate Limit 미들웨어 (RateLimitMiddleware)
  - [x] 429 응답 처리
  - [x] Retry-After 헤더 추가
  - [x] getIPAddress 함수 (X-Forwarded-For, X-Real-IP, RemoteAddr)
  - [x] 모든 테스트 통과 확인 (10개 PASS)

### 6. 라우터 적용
- [x] `cmd/api/main.go` 수정
  - [x] Rate Limiter 초기화 (10 req/min, burst 5)
  - [x] 댓글 작성 엔드포인트에 적용
  - [x] 빌드 성공 확인

### 7. 문서화
- [x] README.md에 Rate Limiting 정책 추가
  - [x] 제한 정책 (10회/분, burst 5)
  - [x] 제한 초과 시 응답
  - [x] IP 추출 방식
- [x] 작업 문서 완료 처리

## 확정된 Rate Limit 정책

### IP 기반 제한
- **댓글 작성**: 10회/분 (burst: 5)
- **댓글 조회**: 적용 안 함 (추후 필요시 100회/분)
- **댓글 수정/삭제**: 10회/분 (작성과 동일)

### 향후 고려사항
- API 키(사이트) 기반: 현재는 IP 기반만 구현
- Redis 기반: 다중 인스턴스 환경 필요 시

---

## 작업 이력

### [2025-10-27] Rate Limiting 구현 완료
- TDD 방식으로 구현 완료
  - `internal/ratelimit/limiter.go` + 테스트 (5개 PASS)
  - `internal/ratelimit/middleware.go` + 테스트 (5개 PASS)
- `internal/httputil` 패키지 생성 (IP/UserAgent 추출 공통화)
- `internal/models/site.go`에 `GenerateAPIKey()` 추가
- 댓글 작성 엔드포인트에 적용 (10 req/min, burst 5)
- README.md 문서화 완료
- 기존 테스트 버그 수정 (api_key NOT NULL, TestGetPostBySlug 등)

### [2025-10-14] 작업 문서 초안 작성
- 러프한 구조 작성
- 구체적인 정책은 작업 시작 시 확정
