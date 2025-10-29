# Admin 인증 시스템 구현 (User, JWT, Google OAuth)

## 작성일
2025-10-29

## 우선순위
- [x] 긴급
- [ ] 높음
- [ ] 보통
- [ ] 낮음

## 작업 개요
Google OAuth 기반 Admin 로그인을 위한 기반 시스템 구현. User 모델, JWT 토큰 관리, Google ID Token 검증 엔드포인트를 TDD 방식으로 개발.

## 작업 목적
- orbithall-admin 프론트엔드에서 Google 로그인을 통해 사용자를 인증
- Google ID Token을 백엔드 자체 JWT로 교환하여 안전한 세션 관리
- 사이트 소유자가 자신의 사이트를 관리할 수 있는 권한 시스템 기반 마련

## 작업 범위

### 포함 사항
- `users`, `user_sites` 테이블 마이그레이션
- User 모델 및 Repository (TDD)
- JWT 생성/검증/파싱 유틸리티 (TDD)
- Google ID Token 검증 라이브러리 통합
- `POST /auth/google/verify` 엔드포인트 (TDD)

### 제외 사항
- JWT 인증 미들웨어 (다음 작업)
- Admin API 엔드포인트 (별도 작업)
- 토큰 갱신(Refresh Token) 로직 (추후)
- 프론트엔드 통합 (orbithall-admin에서 별도)

## 기술적 접근

### 사용할 기술/라이브러리
- **JWT**: `github.com/golang-jwt/jwt/v5`
- **Google OAuth**: `google.golang.org/api/idtoken`
- **database/sql**: 기존 Raw SQL 방식 유지
- **godotenv**: 환경변수 관리 (이미 사용 중)

### 파일 구조
```
orbithall/
├── migrations/
│   ├── 003_create_users_table.up.sql
│   ├── 003_create_users_table.down.sql
│   ├── 004_create_user_sites_table.up.sql
│   └── 004_create_user_sites_table.down.sql
├── internal/
│   ├── models/
│   │   └── user.go
│   ├── database/
│   │   ├── users.go
│   │   └── users_test.go
│   ├── auth/
│   │   ├── jwt.go
│   │   ├── jwt_test.go
│   │   ├── google.go
│   │   └── google_test.go
│   └── handlers/
│       ├── auth.go
│       └── auth_test.go
└── cmd/api/
    └── main.go
```

### 환경변수
```bash
JWT_SECRET=your-secret-key-min-32-chars
JWT_EXPIRATION_HOURS=168
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
```

## 데이터베이스 스키마

### users 테이블
```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    picture_url TEXT,
    google_id VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_id ON users(google_id);
```

### user_sites 테이블
```sql
CREATE TABLE user_sites (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'owner' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, site_id)
);

CREATE INDEX idx_user_sites_user_id ON user_sites(user_id);
CREATE INDEX idx_user_sites_site_id ON user_sites(site_id);
```

## API 명세

### Google ID Token 검증 및 JWT 발급

#### `POST /auth/google/verify`

**요청 본문**
```json
{
  "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6Ij...",
  "email": "user@example.com",
  "name": "홍길동",
  "picture": "https://lh3.googleusercontent.com/..."
}
```

**성공 응답 (200 OK)**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "홍길동",
    "picture_url": "https://lh3.googleusercontent.com/...",
    "created_at": "2025-10-29T10:00:00Z"
  }
}
```

**에러 응답**
- `400`: 입력 검증 실패
- `401`: Google ID Token 검증 실패
- `500`: 서버 오류

## 구현 단계

### 1. 데이터베이스 마이그레이션
**파일**: `migrations/003_create_users_table.up.sql`, `migrations/004_create_user_sites_table.up.sql`

**구현 내용**:
- users 테이블 생성 (email, name, picture_url, google_id)
- google_id UNIQUE 제약 조건
- email, google_id 인덱스
- user_sites 테이블 생성 (다대다 관계)
- 복합 PK (user_id, site_id)
- ON DELETE CASCADE 외래 키

---

### 2. User 모델 구현
**파일**: `internal/models/user.go`

**구현 내용**:
- User 구조체 정의
- JSON 태그 추가 (API 응답용)
- `GoogleID`는 내부 전용 (`json:"-"`)
- ID, Email, Name, PictureURL, CreatedAt, UpdatedAt 필드

---

### 3. User Repository 구현 (TDD)
**파일**: `internal/database/users.go`, `internal/database/users_test.go`

**구현 내용**:
- `CreateUser()`: 새 사용자 생성, RETURNING으로 ID/타임스탬프 반환
- `GetUserByGoogleID()`: Google ID로 사용자 조회, 없으면 nil 반환
- `GetUserByEmail()`: 이메일로 사용자 조회
- `GetUserByID()`: ID로 사용자 조회

**테스트 시나리오**:
- CreateUser 성공 시 ID 설정 확인
- GetUserByGoogleID 존재/없음 케이스
- GetUserByEmail 조회 성공
- 트랜잭션 기반 테스트 (자동 롤백)

---

### 4. JWT 유틸리티 구현 (TDD)
**파일**: `internal/auth/jwt.go`, `internal/auth/jwt_test.go`

**의존성**: `go get github.com/golang-jwt/jwt/v5`

**구현 내용**:
- `CustomClaims` 구조체 (UserID, Email 포함)
- `GenerateJWT()`: HS256 서명, 만료 시간 설정
- `ValidateJWT()`: 토큰 검증, claims 파싱
- 환경변수에서 JWT_SECRET, JWT_EXPIRATION_HOURS 읽기
- Sentinel errors: `ErrInvalidToken`, `ErrExpiredToken`

**주요 로직**:
- JWT_SECRET 최소 32자 검증
- 기본 만료 시간: 168시간 (7일)
- HMAC 서명 방식 확인
- 만료된 토큰은 ErrExpiredToken 반환

**테스트 시나리오**:
- JWT 생성 성공
- 유효한 JWT 검증 (UserID, Email 확인)
- 잘못된 토큰 검증 실패
- 만료된 토큰 검증 실패 (스킵 또는 mock)

---

### 5. Google ID Token 검증 구현
**파일**: `internal/auth/google.go`, `internal/auth/google_test.go`

**의존성**: `go get google.golang.org/api/idtoken`

**구현 내용**:
- `GoogleIDTokenPayload` 구조체 (GoogleID, Email, Name, Picture)
- `VerifyGoogleIDToken()`: Google ID Token 검증
- GOOGLE_CLIENT_ID 환경변수 필수
- Google API idtoken.Validate() 호출
- sub claim에서 Google ID 추출

**주요 로직**:
- idtoken.Validate()로 서명 및 만료 확인
- payload에서 sub(Google ID), email, name, picture 추출
- 검증 실패 시 ErrInvalidIDToken 반환

**테스트**:
- 환경변수 없을 때 에러 확인
- 실제 Google Token 검증은 통합 테스트 또는 모킹 필요

---

### 6. Auth 핸들러 구현 (TDD)
**파일**: `internal/handlers/auth.go`, `internal/handlers/auth_test.go`

**구현 내용**:
- `AuthHandler` 구조체, `NewAuthHandler()` 생성자
- `GoogleVerify()` 핸들러:
  - JSON 요청 본문 파싱
  - 입력 검증 (id_token, email, name 필수)
  - `auth.VerifyGoogleIDToken()` 호출
  - Google ID로 사용자 조회 (`database.GetUserByGoogleID`)
  - 사용자 없으면 생성 (`database.CreateUser`)
  - JWT 생성 (`auth.GenerateJWT`)
  - 200 OK 응답 (token, user)

**주요 로직**:
- 트랜잭션 시작 → 사용자 조회/생성 → 커밋
- Google 검증 실패 시 401 Unauthorized
- 입력 누락 시 400 Bad Request
- DB 오류 시 500 Internal Server Error

**테스트 시나리오**:
- 신규 사용자 생성 및 JWT 발급 (Google 검증 모킹 필요)
- 기존 사용자 로그인
- id_token 누락 시 400
- 잘못된 Google Token 시 401

---

### 7. 라우팅 설정
**파일**: `cmd/api/main.go`

**추가 내용**:
- `authHandler := handlers.NewAuthHandler(db)` 초기화
- `/auth` 라우트 그룹 추가 (인증 불필요)
- `r.Post("/auth/google/verify", authHandler.GoogleVerify)` 등록

**주의**: `/auth` 그룹은 `AuthMiddleware` 적용하지 않음

---

## 검증 방법

### 1. 환경변수 설정
```bash
# .env 파일
JWT_SECRET=your-very-long-secret-key-at-least-32-characters-long
JWT_EXPIRATION_HOURS=168
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
```

### 2. 마이그레이션 실행
```bash
docker exec -it orbithall-api migrate -path /app/migrations -database "$DATABASE_URL" up
```

### 3. 테스트 실행
```bash
go test ./internal/database -run TestCreateUser
go test ./internal/database -run TestGetUserByGoogleID
go test ./internal/auth -run TestGenerateJWT
go test ./internal/auth -run TestValidateJWT
go test ./...
```

### 4. API 수동 테스트
**사전 준비**: Google OAuth Playground에서 실제 ID Token 획득

```bash
curl -X POST http://localhost:8080/auth/google/verify \
  -H "Content-Type: application/json" \
  -d '{
    "id_token": "실제_Google_ID_Token",
    "email": "test@example.com",
    "name": "Test User",
    "picture": "https://example.com/pic.jpg"
  }'

# 예상: 200 OK, JWT 토큰 및 사용자 정보 반환
```

### 5. DB 검증
```sql
-- 사용자 생성 확인
SELECT * FROM users;

-- Google ID 중복 방지 확인
INSERT INTO users (email, name, google_id) VALUES ('dup@example.com', 'Dup', 'google-id-123');
-- 예상: unique constraint 위반 에러
```

## 의존성
- 선행 작업: 없음 (독립적)
- 후속 작업:
  - 010: JWT 인증 미들웨어
  - 011: Site-User 연결 로직
  - 012: Admin API 엔드포인트

## 예상 소요 시간
- 예상: 4-6시간
- 실제: (완료 후 기록)

## 주의사항

### 보안
- ✅ JWT_SECRET은 절대 Git에 커밋하지 않음
- ✅ JWT_SECRET 최소 32자 이상
- ✅ Google ID Token 검증은 서버에서만
- ✅ JWT는 HTTPS에서만 전송 (프로덕션)
- ✅ `google_id`는 JSON 응답에서 제외

### TDD 원칙
- ✅ 테스트 먼저 작성 (Red → Green → Refactor)
- ✅ 트랜잭션 기반 테스트 (자동 롤백)

### Google OAuth
- Google Client ID는 프론트엔드와 동일해야 함
- ID Token은 1시간 후 만료
- 토큰 검증 실패 시 401 반환

## 참고 자료
- Google Identity: https://developers.google.com/identity/sign-in/web/backend-auth
- golang-jwt: https://github.com/golang-jwt/jwt
- Google ID Token: https://pkg.go.dev/google.golang.org/api/idtoken

---

## 작업 이력

### [2025-10-29] 작업 문서 작성
- TDD 기반 구현 단계 정의
- Google OAuth 통합 방식 설계
- User 모델 및 JWT 유틸리티 명세
