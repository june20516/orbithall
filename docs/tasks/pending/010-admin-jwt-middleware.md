# Admin JWT 인증 미들웨어 및 라우팅 분리

## 작성일
2025-10-29

## 우선순위
- [x] 긴급
- [ ] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
JWT 기반 인증 미들웨어를 구현하고, 기존 API Key 인증과 분리된 라우팅 구조를 설계. `/api`는 위젯용 API Key, `/admin`은 JWT 인증을 사용.

## 작업 목적
- Admin 페이지에서 JWT 토큰으로 안전하게 인증
- 기존 위젯 API (API Key)와 Admin API (JWT)의 인증 방식 이분화
- Context에 사용자 정보 저장하여 핸들러에서 접근 가능하도록 함

## 작업 범위

### 포함 사항
- JWT 인증 미들웨어 구현 (TDD)
- Context 헬퍼 함수 (`GetUserFromContext`)
- 라우팅 분리 (`/api`, `/auth`, `/admin`)
- 테스트 (유효한 JWT, 누락, 잘못된 토큰, 만료)

### 제외 사항
- Admin API 엔드포인트 구현 (작업 012)
- Refresh Token 로직 (추후)
- Rate Limiting (별도 작업)

## 기술적 접근

### 사용할 기술/라이브러리
- **Chi 라우터**: 이미 사용 중
- **golang-jwt/jwt/v5**: 작업 009에서 추가됨
- **context**: 사용자 정보 저장

### 파일 구조
```
orbithall/
├── internal/
│   └── handlers/
│       ├── jwt_middleware.go
│       └── jwt_middleware_test.go
└── cmd/api/
    └── main.go
```

## 구현 단계

### 1. JWT 미들웨어 구현 (TDD)

**파일**: `internal/handlers/jwt_middleware.go`, `internal/handlers/jwt_middleware_test.go`

**구현 내용**:
- `JWTAuthMiddleware()`: JWT 기반 인증 미들웨어
  - Authorization 헤더에서 Bearer 토큰 추출
  - "Bearer {token}" 형식 검증
  - `auth.ValidateJWT()` 호출하여 토큰 검증
  - JWT claims에서 UserID 추출
  - `database.GetUserByID()` 호출하여 사용자 조회
  - Context에 사용자 정보 저장
  - 다음 핸들러 호출
- `SetUserInContext()`: Context에 사용자 저장
- `GetUserFromContext()`: Context에서 사용자 추출
- 에러 코드: `MISSING_TOKEN`, `INVALID_TOKEN`, `USER_NOT_FOUND`

**주요 로직**:
- Authorization 헤더 없으면 401
- Bearer 형식 아니면 401
- JWT 검증 실패 시 401
- 사용자 조회 실패 시 401
- 트랜잭션으로 사용자 조회 후 Context 저장

**테스트 시나리오**:
- 유효한 JWT → 통과, Context에 사용자 저장 확인
- JWT 헤더 없음 → 401
- 잘못된 JWT → 401
- 형식 오류 ("Bearer" 없음) → 401
- 존재하지 않는 사용자 ID → 401

---

### 2. 라우팅 분리 (main.go)

**파일**: `cmd/api/main.go`

**변경 내용**:
- 기존 `/api` 그룹: `AuthMiddleware` (API Key) 유지
- 신규 `/auth` 그룹: 인증 불필요 (작업 009에서 추가됨)
- 신규 `/admin` 그룹: `JWTAuthMiddleware` (JWT 인증)

**라우팅 구조**:
```
/health            - 인증 불필요
/auth/*            - 인증 불필요
/api/*             - API Key 인증
/admin/*           - JWT 인증
```

**추가 코드**:
```go
// Admin API (JWT 인증)
r.Route("/admin", func(r chi.Router) {
    r.Use(handlers.JWTAuthMiddleware(db))
    // Admin 엔드포인트는 작업 012에서 추가
})
```

---

## 검증 방법

### 1. 테스트 실행
```bash
go test ./internal/handlers -run TestJWTAuthMiddleware
go test ./...
```

**예상 결과**: 5개 테스트 모두 PASS

---

### 2. API 수동 테스트

**사전 준비**: 작업 009로 JWT 발급
```bash
RESPONSE=$(curl -s -X POST http://localhost:8080/auth/google/verify \
  -H "Content-Type: application/json" \
  -d '{"id_token": "실제_Token", "email": "test@example.com", "name": "Test"}')
JWT=$(echo $RESPONSE | jq -r '.token')
```

**시나리오 1: JWT 없이 Admin API 호출 → 401**
```bash
curl -X GET http://localhost:8080/admin/sites
# 예상: 401 {"error": "MISSING_TOKEN", ...}
```

**시나리오 2: 잘못된 JWT → 401**
```bash
curl -X GET http://localhost:8080/admin/sites \
  -H "Authorization: Bearer invalid-token"
# 예상: 401 {"error": "INVALID_TOKEN", ...}
```

**시나리오 3: 유효한 JWT → 200 (작업 012 완료 후)**
```bash
curl -X GET http://localhost:8080/admin/sites \
  -H "Authorization: Bearer $JWT"
# 예상: 200 OK (작업 012에서 구현)
```

**시나리오 4: 기존 위젯 API는 API Key로 정상 동작**
```bash
curl -X GET http://localhost:8080/api/posts/test/comments \
  -H "X-Orbithall-API-Key: your-api-key"
# 예상: 200 OK (기존 기능 유지)
```

---

### 3. 라우팅 분리 확인
```bash
# /api: API Key 필요
curl http://localhost:8080/api/posts/test/comments
# 예상: 401 MISSING_API_KEY

# /auth: 인증 불필요
curl -X POST http://localhost:8080/auth/google/verify \
  -H "Content-Type: application/json" -d '{}'
# 예상: 400 (입력 오류)

# /admin: JWT 필요
curl http://localhost:8080/admin/sites
# 예상: 401 MISSING_TOKEN

# /health: 인증 불필요
curl http://localhost:8080/health
# 예상: 200 OK
```

---

## 의존성
- 선행 작업: 009 (User, JWT, Google OAuth)
- 후속 작업:
  - 011: Site-User 연결 로직
  - 012: Admin API 엔드포인트

## 예상 소요 시간
- 예상: 2-3시간
- 실제: (완료 후 기록)

## 주의사항

### TDD 원칙
- ✅ 테스트 먼저 작성
- ✅ 트랜잭션 기반 테스트

### JWT 인증
- ✅ Authorization 헤더 형식: `Bearer {token}`
- ✅ 검증 실패 시 401 반환
- ✅ Context에 사용자 정보 저장

### 라우팅 분리
- ✅ `/api`: API Key (기존 유지)
- ✅ `/auth`: 인증 불필요
- ✅ `/admin`: JWT 인증

### 보안
- ✅ JWT는 HTTPS에서만 전송 (프로덕션)
- ✅ 만료된 토큰은 401 반환

## 참고 자료
- Chi Middleware: https://github.com/go-chi/chi#middlewares
- Context Package: https://pkg.go.dev/context

---

## 작업 이력

### [2025-10-29] 작업 문서 작성
- JWT 미들웨어 TDD 구현 계획
- 라우팅 분리 전략 설계
