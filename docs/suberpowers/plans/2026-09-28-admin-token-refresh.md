# 어드민 토큰 갱신 Implementation Plan

> **agentic worker에게:** REQUIRED SUB-SKILL: 이 plan을 task 단위로 구현하려면 suberpower:subagent-driven-development(권장) 또는 suberpower:executing-plans를 사용하세요. Step은 추적을 위해 checkbox(`- [ ]`) 문법을 사용합니다.

**Goal:** 어드민 로그인에 회전(rotation)되는 Refresh Token을 도입하고, `POST /auth/refresh`, `POST /auth/logout`을 제공한다. 인증 관련 에러 본문을 객체 형식으로 통일한다. (명세 7장 전환 계획의 1단계)

**Architecture:** 토큰 암호 연산(생성·해시·HMAC 파생)은 `internal/auth`에, 저장소 쿼리는 `internal/database/refresh_tokens.go`에 둔다. 세션 규칙(발급·회전·유예 시간·재사용 탐지·폐기)은 `internal/handlers/session.go`의 함수들이 조합하고, HTTP 입출력은 `SessionHandler`가 맡는다. 회전은 조건부 UPDATE와 INSERT를 한 SQL 문장으로 처리해 트랜잭션 없이도 원자적이다. 시간 판단은 모두 Go에서 주입한 `now`로 한다(테스트 트랜잭션 안에서는 DB의 `NOW()`가 고정되므로).

**Tech Stack:** Go 1.25, chi v5, PostgreSQL 18(`gen_random_uuid()`), golang-jwt/jwt v5, golang.org/x/time/rate, swaggo(`~/go/bin/swag`)

**Spec:** `docs/specs/admin-auth-token-refresh.md` (v1.3)

**작업 브랜치:** `feat/admin-token-refresh` (Task 0에서 `develop` 기준으로 생성)

**테스트 전제:** `docker compose ps`에서 `orbithall-db`가 healthy여야 한다. 테스트는 로컬 Go로 돌린다. `.env`에 `TEST_DATABASE_URL`이 있어야 DB 통합 테스트가 스킵되지 않는다.

**커밋 메시지:** 기존 이력처럼 `type: 한국어 설명` 형식을 쓰고, 끝에 세션의 attribution 줄을 붙인다.

---

## 파일 구조

| 파일 | 변경 | 책임 |
|---|---|---|
| `docs/tasks/active/021-admin-token-refresh.md` | 생성 → Task 11에서 `completed/`로 이동 | 작업 추적 |
| `migrations/006_create_refresh_tokens_table.{up,down}.sql` | 생성 | `refresh_tokens` 테이블 |
| `internal/models/refresh_token.go` (+ `_test.go`) | 생성 | 저장 모델, 만료 판단, 폐기 사유 상수 |
| `internal/auth/refresh_token.go` (+ `_test.go`) | 생성 | 토큰 생성·해시·후속 토큰 파생·수명 설정 |
| `internal/auth/jwt.go` (+ `jwt_test.go`) | 수정 | `GenerateAccessToken`, `typ`/`iss`/`aud`/`jti` 클레임, `typ` 검증 |
| `internal/database/refresh_tokens.go` (+ `_test.go`) | 생성 | 저장·조회·회전·폐기·정리 쿼리 |
| `internal/handlers/middleware.go` | 수정 (에러 코드 상수 블록) | 인증 에러 코드 상수 |
| `internal/handlers/jwt_middleware.go` (+ `_test.go`) | 수정 | 에러 본문 객체 형식 |
| `internal/handlers/session.go` (+ `session_test.go`) | 생성 | 세션 발급·회전·폐기 규칙 |
| `internal/handlers/auth.go` (+ `auth_test.go`) | 수정 | 로그인 시 세션 발급, 에러 본문 객체 형식 |
| `internal/handlers/session_handler.go` (+ `_test.go`) | 생성 | `/auth/refresh`, `/auth/logout` HTTP |
| `cmd/api/main.go` (+ `main_test.go`) | 수정 | 비밀키 검증, 라우트 등록 |
| `README.md` | 수정 | API·환경변수·Rate Limiting 문서 |
| `docs/tasks/pending/p1-022-admin-token-legacy-cleanup.md` | 생성 | 전환 3단계 후속 작업 |
| `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` | 로컬 생성만 (커밋 안 함, `.gitignore` 대상) | API 문서 확인 |

---

### Task 0: 브랜치와 작업 문서

**Files:**
- Create: `docs/tasks/active/021-admin-token-refresh.md`
- Commit: `docs/specs/admin-auth-token-refresh.md` (이미 작성됨, 미커밋), 이 plan 파일

- [ ] **Step 1: 브랜치 생성 (upstream 없이)**

```bash
git checkout --no-track -b feat/admin-token-refresh develop
git status -sb
```
기대: 첫 줄이 `## feat/admin-token-refresh` (뒤에 `...origin/...`이 없어야 함). 있으면 `git branch --unset-upstream`.

- [ ] **Step 2: 작업 문서 작성**

`docs/tasks/active/021-admin-token-refresh.md`:

```markdown
# [WIP] 어드민 토큰 갱신 (Refresh Token)

## 작성일
2026-09-28

## 우선순위
- [x] 높음

## 작업 개요
어드민 로그인에 회전되는 Refresh Token을 도입해 재로그인 없이 세션을 연장하고, 서버에서 세션을 폐기할 수 있게 한다. 명세 7장 전환 계획의 1단계(백엔드 배포)다.

## 작업 범위
### 포함
- `refresh_tokens` 테이블, `POST /auth/refresh`, `POST /auth/logout`
- 로그인 응답에 토큰 쌍 필드 추가 (`token` 필드는 호환용으로 유지)
- Access Token에 `typ`/`iss`/`aud`/`jti` 클레임 추가, `typ`·`iss`·`aud` 검증 필수 (기존 토큰은 배포 즉시 무효, 재로그인)
- JWT 미들웨어와 `/auth/google/verify`의 에러 본문을 객체 형식으로 통일

### 제외
- 프론트엔드(orbithall-admin) 변경
- `token` 필드 제거 (전환 3단계, p1-022)
- `/admin/*` 핸들러 본문의 평문 에러 통일

## 주요 결정사항
- 회전은 단일 SQL 문장(조건부 UPDATE + INSERT): 트랜잭션 주입 구조에서도 원자성 보장
- 후속 토큰 = HMAC(REFRESH_TOKEN_SECRET, 이전 토큰): 원문 저장 없이 유예 시간 재반환
- rate limit은 계열(family) 기준: 모든 요청이 어드민 서버 IP 하나에서 옴
- 오래된 토큰 정리는 로그인 시: 스케줄러 없이 사용자별로 정리

## 의존성
- 선행: 009, 010
- 후속: 022 (전환 3단계), orbithall-admin 갱신 구현

## 배포 전 필수
- Render 환경변수 `REFRESH_TOKEN_SECRET` 설정 (32자 이상, `JWT_SECRET`과 다른 값)
- 배포 즉시 기존 Access Token이 무효가 되어 로그인 사용자 전원이 재로그인해야 함 (사용자 공지)

---

## 작업 이력
### [2026-09-28] 작업 시작
```

- [ ] **Step 3: Commit**

```bash
git add docs/specs/admin-auth-token-refresh.md docs/suberpowers/plans/2026-09-28-admin-token-refresh.md docs/tasks/active/021-admin-token-refresh.md
git commit -m "docs: 021 어드민 토큰 갱신 명세와 계획"
```

---

### Task 1: `refresh_tokens` 마이그레이션과 모델

**Files:**
- Create: `migrations/006_create_refresh_tokens_table.up.sql`, `migrations/006_create_refresh_tokens_table.down.sql`
- Create: `internal/models/refresh_token.go`
- Test: `internal/models/refresh_token_test.go`

- [ ] **Step 1: 실패하는 모델 테스트 작성**

`internal/models/refresh_token_test.go`:

```go
package models

import (
	"testing"
	"time"
)

// TestRefreshToken_IsExpired는 Refresh Token 만료 판단을 테스트합니다
func TestRefreshToken_IsExpired(t *testing.T) {
	expiresAt := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	token := &RefreshToken{ExpiresAt: expiresAt}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "만료 1초 전이면 유효", now: expiresAt.Add(-time.Second), want: false},
		{name: "만료 시각과 같으면 만료", now: expiresAt, want: true},
		{name: "만료 이후면 만료", now: expiresAt.Add(time.Second), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 만료 여부 확인
			got := token.IsExpired(tt.now)

			// Then: 기대값과 일치
			if got != tt.want {
				t.Errorf("IsExpired(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/models/ -run TestRefreshToken_IsExpired -count=1`
기대: `undefined: RefreshToken`으로 빌드 실패

- [ ] **Step 3: 모델 작성**

`internal/models/refresh_token.go`:

```go
package models

import "time"

// Refresh Token 폐기 사유
const (
	// RefreshTokenRevokedByLogout은 사용자가 로그아웃해 폐기된 경우입니다
	RefreshTokenRevokedByLogout = "logout"

	// RefreshTokenRevokedByReuse는 이미 사용된 토큰이 다시 제출되어 탈취로 판단한 경우입니다
	RefreshTokenRevokedByReuse = "reuse_detected"
)

// RefreshToken은 어드민 로그인 세션을 연장하는 Refresh Token의 저장 정보입니다
// 토큰 원문은 저장하지 않고 SHA-256 해시만 보관합니다
// 서버 내부에서만 쓰며 API 응답으로 내보내지 않습니다
type RefreshToken struct {
	// UserID는 토큰을 소유한 사용자 ID입니다
	UserID int64

	// FamilyID는 한 번의 로그인에서 회전으로 이어진 토큰들이 공유하는 식별자(UUID)입니다
	// 재사용이 탐지되면 같은 FamilyID의 토큰을 모두 폐기합니다
	FamilyID string

	// ParentID는 이 토큰을 만들 때 사용 처리된 이전 토큰의 ID입니다
	// 로그인 시 처음 발급된 토큰은 nil입니다
	ParentID *int64

	// TokenHash는 토큰 원문의 SHA-256 해시입니다
	TokenHash []byte

	// ExpiresAt은 이 토큰의 만료 시각입니다
	// 유휴 만료(발급 시각 + 유휴 수명)와 FamilyExpiresAt 중 이른 시각입니다
	ExpiresAt time.Time

	// FamilyExpiresAt은 계열 전체의 절대 만료 시각입니다 (최초 로그인 시각 + 절대 수명)
	FamilyExpiresAt time.Time

	// UsedAt은 이 토큰으로 회전한 시각입니다 (아직 사용 전이면 nil)
	UsedAt *time.Time

	// RevokedAt은 폐기 시각입니다 (폐기 전이면 nil)
	RevokedAt *time.Time

	// RevokedReason은 폐기 사유입니다 (RefreshTokenRevokedBy* 상수 중 하나)
	RevokedReason *string

	// 메타데이터
	ID        int64
	CreatedAt time.Time
}

// IsExpired는 now 시점에 토큰이 만료되었는지 반환합니다
// ExpiresAt은 항상 FamilyExpiresAt 이하로 저장되므로 ExpiresAt만 비교합니다
func (t *RefreshToken) IsExpired(now time.Time) bool {
	return !now.Before(t.ExpiresAt)
}
```

- [ ] **Step 4: 통과 확인**

실행: `go test ./internal/models/ -run TestRefreshToken_IsExpired -count=1`
기대: `ok`

- [ ] **Step 5: 마이그레이션 작성**

`migrations/006_create_refresh_tokens_table.up.sql`:

```sql
-- refresh_tokens 테이블 생성
-- 어드민 로그인 세션을 연장하는 Refresh Token 저장소
BEGIN;

-- ============================================
-- refresh_tokens 테이블
-- ============================================
-- 토큰 원문은 저장하지 않고 SHA-256 해시만 저장한다
-- 한 번의 로그인에서 회전으로 이어진 토큰들은 같은 family_id를 가진다
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id UUID NOT NULL,
    parent_id BIGINT REFERENCES refresh_tokens(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    family_expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_reason VARCHAR(30),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- 토큰 만료는 계열의 절대 만료를 넘을 수 없다 (IsExpired가 expires_at만 비교하는 전제)
    CONSTRAINT chk_refresh_tokens_expiry CHECK (expires_at <= family_expires_at)
);

-- refresh_tokens 테이블 인덱스
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
-- 부모 토큰 하나에서는 후속 토큰이 하나만 나온다 (계열이 갈라지지 않음을 DB가 보장)
CREATE UNIQUE INDEX idx_refresh_tokens_parent_id ON refresh_tokens(parent_id) WHERE parent_id IS NOT NULL;

COMMIT;
```

`migrations/006_create_refresh_tokens_table.down.sql`:

```sql
-- refresh_tokens 테이블 삭제
BEGIN;

DROP TABLE IF EXISTS refresh_tokens;

COMMIT;
```

- [ ] **Step 6: 마이그레이션 적용 확인**

테스트 DB는 `testhelpers.SetupTestDB`가 테스트 시작 시 마이그레이션을 올린다. 기존 DB 테스트를 한 번 돌려 006이 문법 오류 없이 적용되는지 확인한다.

실행: `go test ./internal/database/ -run TestCreateUser -count=1`
기대: `ok` (마이그레이션 실패 시 `Failed to run migrations`로 FAIL)

실행: `docker compose exec postgres psql -U orbithall -d test_orbithall_db -c '\d refresh_tokens'`
기대: 위 컬럼과 인덱스 3개(`family_id`, `user_id`, `parent_id` 부분 unique)가 보임

- [ ] **Step 7: Commit**

```bash
git add migrations/006_create_refresh_tokens_table.up.sql migrations/006_create_refresh_tokens_table.down.sql internal/models/refresh_token.go internal/models/refresh_token_test.go
git commit -m "feat: refresh_tokens 테이블과 모델 추가"
```

---

### Task 2: Refresh Token 생성·해시·파생 (`internal/auth`)

**Files:**
- Create: `internal/auth/refresh_token.go`
- Test: `internal/auth/refresh_token_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/auth/refresh_token_test.go`:

```go
package auth

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// testRefreshTokenSecret은 테스트용 Refresh Token 비밀키입니다 (jwt_test.go의 JWT_SECRET과 다른 값)
const testRefreshTokenSecret = "test-refresh-secret-at-least-32-characters-long"

// TestGenerateRefreshToken은 로그인 시 첫 Refresh Token 생성을 테스트합니다
func TestGenerateRefreshToken(t *testing.T) {
	t.Run("접두사가 붙은 base64url 문자열을 만든다", func(t *testing.T) {
		// When: 토큰 생성
		token, err := GenerateRefreshToken()

		// Then: 접두사 + 32바이트의 패딩 없는 base64url(43자)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if !strings.HasPrefix(token, RefreshTokenPrefix) {
			t.Errorf("expected prefix %q, got %q", RefreshTokenPrefix, token)
		}
		if len(token) != len(RefreshTokenPrefix)+43 {
			t.Errorf("expected length %d, got %d", len(RefreshTokenPrefix)+43, len(token))
		}
	})

	t.Run("호출할 때마다 다른 값을 만든다", func(t *testing.T) {
		// When: 두 번 생성
		first, _ := GenerateRefreshToken()
		second, _ := GenerateRefreshToken()

		// Then: 서로 다름
		if first == second {
			t.Error("expected different tokens")
		}
	})
}

// TestDeriveNextRefreshToken은 회전 시 후속 토큰 파생을 테스트합니다
func TestDeriveNextRefreshToken(t *testing.T) {
	t.Setenv("REFRESH_TOKEN_SECRET", testRefreshTokenSecret)

	t.Run("같은 입력이면 같은 후속 토큰을 만든다", func(t *testing.T) {
		// When: 같은 토큰에서 두 번 파생
		first, err := DeriveNextRefreshToken("ohrt_same-input")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		second, _ := DeriveNextRefreshToken("ohrt_same-input")

		// Then: 결과가 같음
		if first != second {
			t.Errorf("expected deterministic result, got %q and %q", first, second)
		}
	})

	t.Run("입력이 다르면 다른 후속 토큰을 만든다", func(t *testing.T) {
		// When: 서로 다른 토큰에서 파생
		first, _ := DeriveNextRefreshToken("ohrt_input-a")
		second, _ := DeriveNextRefreshToken("ohrt_input-b")

		// Then: 결과가 다름
		if first == second {
			t.Error("expected different results for different inputs")
		}
	})

	t.Run("입력과 다르고 접두사가 붙는다", func(t *testing.T) {
		// When: 파생
		next, _ := DeriveNextRefreshToken("ohrt_input")

		// Then: 입력과 다르고 접두사 유지
		if next == "ohrt_input" {
			t.Error("expected derived token to differ from input")
		}
		if !strings.HasPrefix(next, RefreshTokenPrefix) {
			t.Errorf("expected prefix %q, got %q", RefreshTokenPrefix, next)
		}
	})
}

// TestDeriveNextRefreshToken_DependsOnSecret은 비밀키가 바뀌면 파생 결과도 바뀌는지 테스트합니다
func TestDeriveNextRefreshToken_DependsOnSecret(t *testing.T) {
	// Given: 비밀키 A로 파생
	t.Setenv("REFRESH_TOKEN_SECRET", testRefreshTokenSecret)
	withSecretA, _ := DeriveNextRefreshToken("ohrt_input")

	// When: 비밀키 B로 파생
	t.Setenv("REFRESH_TOKEN_SECRET", testRefreshTokenSecret+"-other")
	withSecretB, _ := DeriveNextRefreshToken("ohrt_input")

	// Then: 결과가 다름
	if withSecretA == withSecretB {
		t.Error("expected different results for different secrets")
	}
}

// TestValidateRefreshTokenSecret은 비밀키 검증을 테스트합니다
func TestValidateRefreshTokenSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{name: "비어 있으면 에러", secret: "", wantErr: true},
		{name: "32자 미만이면 에러", secret: "too-short-secret", wantErr: true},
		{name: "JWT_SECRET과 같으면 에러", secret: os.Getenv("JWT_SECRET"), wantErr: true},
		{name: "32자 이상이고 JWT_SECRET과 다르면 통과", secret: testRefreshTokenSecret, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 비밀키 설정
			t.Setenv("REFRESH_TOKEN_SECRET", tt.secret)

			// When: 검증
			err := ValidateRefreshTokenSecret()

			// Then: 기대한 에러 여부
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRefreshTokenSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDeriveNextRefreshToken_InvalidSecret은 비밀키가 잘못되면 파생이 실패하는지 테스트합니다
func TestDeriveNextRefreshToken_InvalidSecret(t *testing.T) {
	// Given: 너무 짧은 비밀키
	t.Setenv("REFRESH_TOKEN_SECRET", "short")

	// When: 파생
	_, err := DeriveNextRefreshToken("ohrt_input")

	// Then: 에러
	if err == nil {
		t.Fatal("expected error for invalid secret, got nil")
	}
}

// TestHashRefreshToken은 저장용 해시 계산을 테스트합니다
func TestHashRefreshToken(t *testing.T) {
	// When: 같은 입력과 다른 입력의 해시 계산
	first := HashRefreshToken("ohrt_value")
	second := HashRefreshToken("ohrt_value")
	other := HashRefreshToken("ohrt_other")

	// Then: SHA-256 길이(32바이트), 결정적, 입력별로 다름
	if len(first) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(first))
	}
	if !bytes.Equal(first, second) {
		t.Error("expected same hash for same input")
	}
	if bytes.Equal(first, other) {
		t.Error("expected different hash for different input")
	}
}

// TestLoadRefreshTokenConfig는 수명 설정 로드를 테스트합니다
func TestLoadRefreshTokenConfig(t *testing.T) {
	t.Run("환경변수가 없으면 기본값", func(t *testing.T) {
		// Given: 환경변수 비움
		t.Setenv("REFRESH_TOKEN_IDLE_TTL", "")
		t.Setenv("REFRESH_TOKEN_ABSOLUTE_TTL", "")
		t.Setenv("REFRESH_TOKEN_REUSE_GRACE", "")

		// When: 로드
		cfg := LoadRefreshTokenConfig()

		// Then: 14일, 30일, 30초
		if cfg.IdleTTL != 14*24*time.Hour {
			t.Errorf("IdleTTL = %v", cfg.IdleTTL)
		}
		if cfg.AbsoluteTTL != 30*24*time.Hour {
			t.Errorf("AbsoluteTTL = %v", cfg.AbsoluteTTL)
		}
		if cfg.ReuseGrace != 30*time.Second {
			t.Errorf("ReuseGrace = %v", cfg.ReuseGrace)
		}
	})

	t.Run("환경변수 값을 time.Duration 형식으로 읽는다", func(t *testing.T) {
		// Given: 값 설정
		t.Setenv("REFRESH_TOKEN_IDLE_TTL", "48h")
		t.Setenv("REFRESH_TOKEN_ABSOLUTE_TTL", "96h")
		t.Setenv("REFRESH_TOKEN_REUSE_GRACE", "10s")

		// When: 로드
		cfg := LoadRefreshTokenConfig()

		// Then: 설정값 반영
		if cfg.IdleTTL != 48*time.Hour || cfg.AbsoluteTTL != 96*time.Hour || cfg.ReuseGrace != 10*time.Second {
			t.Errorf("unexpected config: %+v", cfg)
		}
	})

	t.Run("형식이 잘못되었거나 0 이하이면 기본값", func(t *testing.T) {
		// Given: 잘못된 값
		t.Setenv("REFRESH_TOKEN_IDLE_TTL", "two weeks")
		t.Setenv("REFRESH_TOKEN_ABSOLUTE_TTL", "-1h")
		t.Setenv("REFRESH_TOKEN_REUSE_GRACE", "0s")

		// When: 로드
		cfg := LoadRefreshTokenConfig()

		// Then: 기본값
		if cfg.IdleTTL != 14*24*time.Hour || cfg.AbsoluteTTL != 30*24*time.Hour || cfg.ReuseGrace != 30*time.Second {
			t.Errorf("expected defaults, got: %+v", cfg)
		}
	})
}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/auth/ -count=1`
기대: `undefined: GenerateRefreshToken` 등으로 빌드 실패

- [ ] **Step 3: 구현**

`internal/auth/refresh_token.go`:

```go
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

// RefreshTokenPrefix는 Refresh Token 앞에 붙는 식별 접두사입니다
// 토큰이 로그나 저장소에 유출되었을 때 스캐너가 쉽게 찾을 수 있게 합니다
const RefreshTokenPrefix = "ohrt_"

// refreshTokenRandomBytes는 로그인 시 첫 Refresh Token을 만드는 난수 길이입니다
const refreshTokenRandomBytes = 32

// Refresh Token 수명 기본값
const (
	defaultRefreshTokenIdleTTL     = 14 * 24 * time.Hour
	defaultRefreshTokenAbsoluteTTL = 30 * 24 * time.Hour
	defaultRefreshTokenReuseGrace  = 30 * time.Second
)

// RefreshTokenConfig는 Refresh Token 수명 설정입니다
type RefreshTokenConfig struct {
	// IdleTTL은 토큰 발급 후 이 시간 동안 쓰지 않으면 만료되는 기간입니다
	IdleTTL time.Duration

	// AbsoluteTTL은 최초 로그인 후 갱신과 관계없이 세션이 끝나는 기간입니다
	AbsoluteTTL time.Duration

	// ReuseGrace는 사용된 토큰이 다시 제출되어도 재사용으로 보지 않는 유예 시간입니다
	// 동시 요청이나 쿠키 저장 실패로 같은 토큰이 두 번 오는 경우를 흡수합니다
	ReuseGrace time.Duration
}

// LoadRefreshTokenConfig는 환경변수에서 Refresh Token 수명 설정을 읽습니다
// 값은 time.ParseDuration 형식(예: "336h", "30s")이며, 없거나 잘못되면 기본값을 씁니다
func LoadRefreshTokenConfig() RefreshTokenConfig {
	return RefreshTokenConfig{
		IdleTTL:     durationFromEnv("REFRESH_TOKEN_IDLE_TTL", defaultRefreshTokenIdleTTL),
		AbsoluteTTL: durationFromEnv("REFRESH_TOKEN_ABSOLUTE_TTL", defaultRefreshTokenAbsoluteTTL),
		ReuseGrace:  durationFromEnv("REFRESH_TOKEN_REUSE_GRACE", defaultRefreshTokenReuseGrace),
	}
}

// durationFromEnv는 환경변수를 time.Duration으로 읽고, 없거나 0 이하이거나 형식이 틀리면 fallback을 반환합니다
func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

// GenerateRefreshToken은 로그인 시 발급하는 첫 Refresh Token을 만듭니다
// 암호학적 난수 32바이트를 패딩 없는 base64url로 인코딩하고 접두사를 붙입니다
func GenerateRefreshToken() (string, error) {
	randomBytes := make([]byte, refreshTokenRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

// DeriveNextRefreshToken은 회전 시 이전 토큰에서 후속 토큰을 만듭니다
// HMAC-SHA256(REFRESH_TOKEN_SECRET, 이전 토큰 원문)이므로 같은 이전 토큰에서는 항상 같은 후속 토큰이 나옵니다
// 덕분에 원문을 저장하지 않고도 유예 시간 안의 재요청에 같은 후속 토큰을 다시 돌려줄 수 있습니다
// 비밀키 없이는 이전 토큰으로 후속 토큰을 예측할 수 없습니다
func DeriveNextRefreshToken(current string) (string, error) {
	secret, err := refreshTokenSecret()
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(current))

	return RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// ValidateRefreshTokenSecret은 REFRESH_TOKEN_SECRET 환경변수가 올바른지 확인합니다
// 서버 시작 시 호출해 설정 누락을 조기에 발견합니다
func ValidateRefreshTokenSecret() error {
	_, err := refreshTokenSecret()
	return err
}

// refreshTokenSecret은 후속 토큰 파생에 쓰는 비밀키를 읽고 검증합니다
// JWT_SECRET과 같은 값을 쓰면 한 키의 유출이 두 용도 모두에 영향을 주므로 거부합니다
func refreshTokenSecret() ([]byte, error) {
	secret := os.Getenv("REFRESH_TOKEN_SECRET")
	if len(secret) < 32 {
		return nil, fmt.Errorf("REFRESH_TOKEN_SECRET must be at least 32 characters long")
	}
	if secret == os.Getenv("JWT_SECRET") {
		return nil, fmt.Errorf("REFRESH_TOKEN_SECRET must differ from JWT_SECRET")
	}

	return []byte(secret), nil
}

// HashRefreshToken은 DB에 저장하고 조회할 때 쓰는 토큰의 SHA-256 해시를 반환합니다
// 토큰 자체가 충분한 난수이므로 salt 없는 단일 해시로 충분합니다
func HashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
```

- [ ] **Step 4: 통과 확인**

실행: `go test ./internal/auth/ -count=1`
기대: `ok` (Google 관련 테스트가 환경변수 때문에 스킵되는 것은 기존과 동일)

- [ ] **Step 5: Commit**

```bash
git add internal/auth/refresh_token.go internal/auth/refresh_token_test.go
git commit -m "feat: Refresh Token 생성·해시·후속 토큰 파생 추가"
```

---

### Task 3: Access Token 클레임 필수화와 만료 시각 반환 (`internal/auth/jwt.go`)

**Files:**
- Modify: `internal/auth/jwt.go` (`CustomClaims`, `GenerateJWT`, `ValidateJWT`)
- Test: `internal/auth/jwt_test.go` (파일 끝에 추가)

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/auth/jwt_test.go` 끝에 추가한다. import에 `"github.com/golang-jwt/jwt/v5"`가 없으면 추가한다(`os`, `testing`, `time`은 이미 있음).

```go
// signTestClaims는 테스트용 클레임을 JWT_SECRET으로 서명합니다
func signTestClaims(t *testing.T, claims *CustomClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		t.Fatalf("failed to sign test claims: %v", err)
	}
	return token
}

// TestGenerateAccessToken은 Access Token 발급 시 클레임과 만료 시각을 테스트합니다
func TestGenerateAccessToken(t *testing.T) {
	t.Run("Access Token 클레임과 만료 시각을 함께 반환한다", func(t *testing.T) {
		// Given: 발급 전 시각
		before := time.Now()

		// When: Access Token 발급
		token, expiresAt, err := GenerateAccessToken(42, "access@example.com")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		claims, err := ValidateJWT(token)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		// Then: typ/iss/aud/jti 클레임이 있고 exp가 반환값과 같음
		if claims.TokenType != AccessTokenType {
			t.Errorf("typ = %q, want %q", claims.TokenType, AccessTokenType)
		}
		if claims.Issuer != TokenIssuer {
			t.Errorf("iss = %q, want %q", claims.Issuer, TokenIssuer)
		}
		if len(claims.Audience) != 1 || claims.Audience[0] != AdminAudience {
			t.Errorf("aud = %v, want [%q]", claims.Audience, AdminAudience)
		}
		if claims.ID == "" {
			t.Error("expected non-empty jti")
		}
		if !claims.ExpiresAt.Time.Equal(expiresAt) {
			t.Errorf("exp = %v, returned expiresAt = %v", claims.ExpiresAt.Time, expiresAt)
		}

		// Then: 만료 시각은 발급 시각 + JWT_EXPIRATION_HOURS(테스트 init에서 168)
		earliest := before.Add(168 * time.Hour).Add(-time.Second)
		latest := time.Now().Add(168 * time.Hour)
		if expiresAt.Before(earliest) || expiresAt.After(latest) {
			t.Errorf("expiresAt %v not within [%v, %v]", expiresAt, earliest, latest)
		}
	})

	t.Run("발급할 때마다 jti가 다르다", func(t *testing.T) {
		// When: 같은 사용자로 두 번 발급
		first, _, _ := GenerateAccessToken(42, "access@example.com")
		second, _, _ := GenerateAccessToken(42, "access@example.com")
		firstClaims, _ := ValidateJWT(first)
		secondClaims, _ := ValidateJWT(second)

		// Then: jti가 다름
		if firstClaims.ID == secondClaims.ID {
			t.Error("expected different jti values")
		}
	})
}

// validTestClaims는 필수 클레임을 모두 갖춘 Access Token 클레임입니다
func validTestClaims() *CustomClaims {
	return &CustomClaims{
		UserID:    7,
		Email:     "claims@example.com",
		TokenType: AccessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Audience:  jwt.ClaimStrings{AdminAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
}

// TestValidateJWT_RequiredClaims는 typ·iss·aud 필수 검증을 테스트합니다
func TestValidateJWT_RequiredClaims(t *testing.T) {
	t.Run("필수 클레임이 모두 맞으면 통과한다", func(t *testing.T) {
		// Given: 올바른 클레임
		token := signTestClaims(t, validTestClaims())

		// When: 검증
		claims, err := ValidateJWT(token)

		// Then: 통과
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if claims.UserID != 7 {
			t.Errorf("UserID = %d, want 7", claims.UserID)
		}
	})

	tests := []struct {
		name   string
		mutate func(claims *CustomClaims)
	}{
		{name: "typ가 없으면 거부", mutate: func(c *CustomClaims) { c.TokenType = "" }},
		{name: "typ가 access가 아니면 거부", mutate: func(c *CustomClaims) { c.TokenType = "refresh" }},
		{name: "iss가 없으면 거부", mutate: func(c *CustomClaims) { c.Issuer = "" }},
		{name: "iss가 다르면 거부", mutate: func(c *CustomClaims) { c.Issuer = "someone-else" }},
		{name: "aud가 없으면 거부", mutate: func(c *CustomClaims) { c.Audience = nil }},
		{name: "aud가 다르면 거부", mutate: func(c *CustomClaims) { c.Audience = jwt.ClaimStrings{"other-service"} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 필수 클레임 하나가 잘못된 토큰
			claims := validTestClaims()
			tt.mutate(claims)
			token := signTestClaims(t, claims)

			// When: 검증
			_, err := ValidateJWT(token)

			// Then: ErrInvalidToken
			if err != ErrInvalidToken {
				t.Errorf("expected ErrInvalidToken, got: %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/auth/ -run 'TestGenerateAccessToken|TestValidateJWT_RequiredClaims' -count=1`
기대: `undefined: GenerateAccessToken`, `claims.TokenType undefined` 등으로 빌드 실패

- [ ] **Step 3: 구현**

`internal/auth/jwt.go`를 다음처럼 바꾼다.

import 블록에 `"crypto/rand"`를 추가한다.

`var (...)` 에러 블록 아래에 상수를 추가한다.

```go
const (
	// AccessTokenType은 어드민 API 호출용 Access Token의 typ 클레임 값입니다
	AccessTokenType = "access"

	// TokenIssuer는 이 서버가 발급한 토큰임을 나타내는 iss 클레임 값입니다
	TokenIssuer = "orbithall"

	// AdminAudience는 토큰을 받는 대상(어드민 API)을 나타내는 aud 클레임 값입니다
	AdminAudience = "orbithall-admin"
)
```

`CustomClaims`를 다음으로 바꾼다.

```go
// CustomClaims는 JWT 토큰에 포함될 사용자 정의 클레임입니다
type CustomClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`

	// TokenType은 토큰 용도입니다 (Access Token은 AccessTokenType)
	TokenType string `json:"typ,omitempty"`

	jwt.RegisteredClaims
}
```

기존 `GenerateJWT` 함수 전체(주석 포함)를 다음 두 함수로 바꾼다.

```go
// GenerateJWT는 Access Token을 발급하고 토큰 문자열만 반환합니다
// 만료 시각이 필요하면 GenerateAccessToken을 사용합니다
func GenerateJWT(userID int64, email string) (string, error) {
	token, _, err := GenerateAccessToken(userID, email)
	return token, err
}

// GenerateAccessToken은 사용자 ID와 이메일을 담은 Access Token과 그 만료 시각을 반환합니다
// JWT_SECRET과 JWT_EXPIRATION_HOURS 환경변수를 사용합니다
func GenerateAccessToken(userID int64, email string) (string, time.Time, error) {
	// JWT_SECRET 검증
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return "", time.Time{}, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if len(jwtSecret) < 32 {
		return "", time.Time{}, fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	// JWT_EXPIRATION_HOURS 읽기 (기본값: 168시간 = 7일)
	expirationHours := 168
	if expirationStr := os.Getenv("JWT_EXPIRATION_HOURS"); expirationStr != "" {
		if parsed, err := strconv.Atoi(expirationStr); err == nil {
			expirationHours = parsed
		}
	}

	// 만료 시간 계산
	// JWT의 exp는 초 단위로 저장되므로 반환값도 초 단위로 맞춥니다
	now := time.Now()
	expiresAt := now.Add(time.Duration(expirationHours) * time.Hour).Truncate(time.Second)

	// Claims 생성
	claims := &CustomClaims{
		UserID:    userID,
		Email:     email,
		TokenType: AccessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Audience:  jwt.ClaimStrings{AdminAudience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			// jti: 토큰마다 고유한 식별자
			ID: rand.Text(),
		},
	}

	// JWT 토큰 생성 (HS256 알고리즘)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 서명하여 문자열로 변환
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}
```

`ValidateJWT`의 `jwt.ParseWithClaims` 호출에 iss·aud 검증 옵션을 넘긴다. 키 함수 뒤에 인자 두 개를 추가하고, 호출 위에 주석을 단다.

```go
	// 토큰 파싱 및 검증
	// iss와 aud가 없거나 이 서버·어드민 API용 값이 아니면 거부합니다
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// HMAC 서명 방식인지 확인
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	}, jwt.WithIssuer(TokenIssuer), jwt.WithAudience(AdminAudience))
```

그리고 `ValidateJWT`의 끝부분(`// Claims 추출` 블록 다음, `return claims, nil` 앞)에 추가한다.

```go
	// Access Token 용도의 토큰만 받습니다
	if claims.TokenType != AccessTokenType {
		return nil, ErrInvalidToken
	}
```

만료 판단은 그대로 `errors.Is(err, jwt.ErrTokenExpired)`로 한다. golang-jwt v5는 검증 에러를 모두 모아 반환하므로, 다른 클레임 오류와 함께 만료되어도 `ErrExpiredToken`이 반환된다.

- [ ] **Step 4: 통과 확인**

실행: `go test ./internal/auth/ -count=1`
기대: `ok` (기존 JWT 테스트는 `GenerateJWT`로 토큰을 만들므로 새 클레임을 모두 가짐)

실행: `go build ./...`
기대: 에러 없음 (`GenerateJWT` 시그니처는 그대로라 호출부 변경 없음)

- [ ] **Step 5: Commit**

```bash
git add internal/auth/jwt.go internal/auth/jwt_test.go
git commit -m "feat: Access Token 클레임(typ·iss·aud·jti) 추가와 검증 필수화"
```

---

### Task 4: Refresh Token 저장소 (`internal/database/refresh_tokens.go`)

**Files:**
- Create: `internal/database/refresh_tokens.go`
- Test: `internal/database/refresh_tokens_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/database/refresh_tokens_test.go`:

```go
package database

import (
	"context"
	"testing"
	"time"

	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

// refreshTokenTestTime은 저장소 테스트의 기준 시각입니다
var refreshTokenTestTime = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// createRefreshTokenTestUser는 Refresh Token 테스트용 사용자를 만듭니다
func createRefreshTokenTestUser(ctx context.Context, t *testing.T, tx DBTX) *models.User {
	t.Helper()
	user := &models.User{
		Email:    "refresh-token@example.com",
		Name:     "Refresh Token User",
		GoogleID: "google-refresh-token-user",
	}
	if err := CreateUser(ctx, tx, user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// createTestRefreshTokenFamily는 hash 값으로 새 계열의 첫 토큰을 저장합니다
func createTestRefreshTokenFamily(ctx context.Context, t *testing.T, tx DBTX, userID int64, hash string, familyExpiresAt time.Time) *models.RefreshToken {
	t.Helper()
	token, err := CreateRefreshTokenFamily(ctx, tx, userID, []byte(hash), familyExpiresAt, familyExpiresAt)
	if err != nil {
		t.Fatalf("failed to create refresh token family: %v", err)
	}
	return token
}

// TestCreateRefreshTokenFamily는 로그인 시 첫 토큰 저장을 테스트합니다
func TestCreateRefreshTokenFamily(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("새 계열의 첫 토큰을 저장한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자와 만료 시각
		user := createRefreshTokenTestUser(ctx, t, tx)
		expiresAt := refreshTokenTestTime.Add(14 * 24 * time.Hour)
		familyExpiresAt := refreshTokenTestTime.Add(30 * 24 * time.Hour)

		// When: 저장
		token, err := CreateRefreshTokenFamily(ctx, tx, user.ID, []byte("hash-create"), expiresAt, familyExpiresAt)

		// Then: 계열 ID가 생기고 부모·사용·폐기 정보는 비어 있음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if token.ID == 0 || token.FamilyID == "" {
			t.Errorf("expected ID and FamilyID, got %+v", token)
		}
		if token.UserID != user.ID {
			t.Errorf("UserID = %d, want %d", token.UserID, user.ID)
		}
		if token.ParentID != nil || token.UsedAt != nil || token.RevokedAt != nil {
			t.Errorf("expected nil parent/used/revoked, got %+v", token)
		}
		if !token.ExpiresAt.Equal(expiresAt) || !token.FamilyExpiresAt.Equal(familyExpiresAt) {
			t.Errorf("unexpected expiry: %v / %v", token.ExpiresAt, token.FamilyExpiresAt)
		}
	})

	t.Run("로그인마다 다른 계열 ID를 가진다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자
		user := createRefreshTokenTestUser(ctx, t, tx)
		familyExpiresAt := refreshTokenTestTime.Add(time.Hour)

		// When: 두 번 로그인
		first := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-family-1", familyExpiresAt)
		second := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-family-2", familyExpiresAt)

		// Then: 계열 ID가 다름
		if first.FamilyID == second.FamilyID {
			t.Error("expected different family IDs")
		}
	})
}

// TestGetRefreshTokenByHash는 해시로 토큰 조회를 테스트합니다
func TestGetRefreshTokenByHash(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("저장된 토큰을 찾는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 저장된 토큰
		user := createRefreshTokenTestUser(ctx, t, tx)
		created := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-get", refreshTokenTestTime.Add(time.Hour))

		// When: 해시로 조회
		found, err := GetRefreshTokenByHash(ctx, tx, []byte("hash-get"))

		// Then: 같은 토큰
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if found == nil || found.ID != created.ID {
			t.Errorf("expected token %d, got %+v", created.ID, found)
		}
	})

	t.Run("없으면 nil을 반환한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 없는 해시로 조회
		found, err := GetRefreshTokenByHash(ctx, tx, []byte("hash-missing"))

		// Then: nil, nil
		if err != nil || found != nil {
			t.Errorf("expected nil, nil; got %+v, %v", found, err)
		}
	})
}

// TestRotateRefreshToken은 회전(이전 토큰 사용 처리 + 후속 토큰 저장)을 테스트합니다
func TestRotateRefreshToken(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("이전 토큰을 사용 처리하고 후속 토큰을 저장한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 첫 토큰
		user := createRefreshTokenTestUser(ctx, t, tx)
		familyExpiresAt := refreshTokenTestTime.Add(30 * 24 * time.Hour)
		parent := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-parent", familyExpiresAt)
		childExpiresAt := refreshTokenTestTime.Add(14 * 24 * time.Hour)

		// When: 회전
		child, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-child"), childExpiresAt, refreshTokenTestTime)

		// Then: 후속 토큰이 같은 계열·절대 만료를 이어받고 부모를 가리킴
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if child == nil {
			t.Fatal("expected child token, got nil")
		}
		if child.ParentID == nil || *child.ParentID != parent.ID {
			t.Errorf("ParentID = %v, want %d", child.ParentID, parent.ID)
		}
		if child.FamilyID != parent.FamilyID || !child.FamilyExpiresAt.Equal(familyExpiresAt) {
			t.Errorf("expected same family, got %+v", child)
		}
		if !child.ExpiresAt.Equal(childExpiresAt) {
			t.Errorf("ExpiresAt = %v, want %v", child.ExpiresAt, childExpiresAt)
		}

		// Then: 부모는 사용 처리됨
		reloaded, _ := GetRefreshTokenByHash(ctx, tx, []byte("hash-parent"))
		if reloaded.UsedAt == nil || !reloaded.UsedAt.Equal(refreshTokenTestTime) {
			t.Errorf("parent UsedAt = %v, want %v", reloaded.UsedAt, refreshTokenTestTime)
		}
	})

	t.Run("이미 사용된 토큰은 다시 회전되지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 한 번 회전한 토큰
		user := createRefreshTokenTestUser(ctx, t, tx)
		parent := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-used", refreshTokenTestTime.Add(time.Hour))
		if _, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-used-child"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime); err != nil {
			t.Fatalf("first rotation failed: %v", err)
		}

		// When: 같은 부모로 다시 회전
		second, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-used-child-2"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime)

		// Then: nil, nil
		if err != nil || second != nil {
			t.Errorf("expected nil, nil; got %+v, %v", second, err)
		}
	})

	t.Run("폐기된 토큰은 회전되지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 폐기된 토큰
		user := createRefreshTokenTestUser(ctx, t, tx)
		parent := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-revoked", refreshTokenTestTime.Add(time.Hour))
		if err := RevokeRefreshTokenFamily(ctx, tx, parent.FamilyID, models.RefreshTokenRevokedByLogout, refreshTokenTestTime); err != nil {
			t.Fatalf("failed to revoke: %v", err)
		}

		// When: 회전
		child, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-revoked-child"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime)

		// Then: nil, nil
		if err != nil || child != nil {
			t.Errorf("expected nil, nil; got %+v, %v", child, err)
		}
	})
}

// TestRotateRefreshToken_FamilyRevoked는 계열 폐기를 빠져나간 토큰이 회전되지 않는지 테스트합니다
// 폐기와 동시에 진행된 회전이 만든 후속 토큰은 revoked_at이 비어 있을 수 있으므로, 같은 계열의 폐기 여부로 막아야 합니다
func TestRotateRefreshToken_FamilyRevoked(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// Given: 폐기된 계열과, 같은 계열에서 폐기되지 않고 남은 토큰(동시 회전으로 폐기를 빠져나간 상황)
	user := createRefreshTokenTestUser(ctx, t, tx)
	familyExpiresAt := refreshTokenTestTime.Add(time.Hour)
	first := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-family-first", familyExpiresAt)
	if err := RevokeRefreshTokenFamily(ctx, tx, first.FamilyID, models.RefreshTokenRevokedByReuse, refreshTokenTestTime); err != nil {
		t.Fatalf("failed to revoke: %v", err)
	}
	var escapedID int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO refresh_tokens (user_id, family_id, parent_id, token_hash, expires_at, family_expires_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id
	`, user.ID, first.FamilyID, first.ID, []byte("hash-escaped"), familyExpiresAt).Scan(&escapedID)
	if err != nil {
		t.Fatalf("failed to insert escaped token: %v", err)
	}

	// When: 남은 토큰으로 회전
	child, err := RotateRefreshToken(ctx, tx, escapedID, []byte("hash-escaped-child"), familyExpiresAt, refreshTokenTestTime)

	// Then: nil, nil이고 남은 토큰은 사용 처리되지 않음
	if err != nil || child != nil {
		t.Errorf("expected nil, nil; got %+v, %v", child, err)
	}
	escaped, _ := GetRefreshTokenByHash(ctx, tx, []byte("hash-escaped"))
	if escaped.UsedAt != nil {
		t.Errorf("expected escaped token to stay unused, got UsedAt %v", escaped.UsedAt)
	}
}

// TestGetChildRefreshToken은 후속 토큰 조회를 테스트합니다
func TestGetChildRefreshToken(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// Given: 회전하지 않은 토큰과 회전한 토큰
	user := createRefreshTokenTestUser(ctx, t, tx)
	lonely := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-lonely", refreshTokenTestTime.Add(time.Hour))
	parent := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-with-child", refreshTokenTestTime.Add(time.Hour))
	child, err := RotateRefreshToken(ctx, tx, parent.ID, []byte("hash-the-child"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime)
	if err != nil {
		t.Fatalf("rotation failed: %v", err)
	}

	// When: 후속 토큰 조회
	foundChild, err := GetChildRefreshToken(ctx, tx, parent.ID)
	noChild, noChildErr := GetChildRefreshToken(ctx, tx, lonely.ID)

	// Then: 회전한 토큰은 후속 토큰을, 아닌 토큰은 nil을 반환
	if err != nil || foundChild == nil || foundChild.ID != child.ID {
		t.Errorf("expected child %d, got %+v, %v", child.ID, foundChild, err)
	}
	if noChildErr != nil || noChild != nil {
		t.Errorf("expected nil, nil; got %+v, %v", noChild, noChildErr)
	}
}

// TestRevokeRefreshTokenFamily는 계열 폐기를 테스트합니다
func TestRevokeRefreshTokenFamily(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// Given: 토큰 두 개로 이어진 계열 A와 별도 계열 B
	user := createRefreshTokenTestUser(ctx, t, tx)
	familyA := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-a1", refreshTokenTestTime.Add(time.Hour))
	if _, err := RotateRefreshToken(ctx, tx, familyA.ID, []byte("hash-a2"), refreshTokenTestTime.Add(time.Hour), refreshTokenTestTime); err != nil {
		t.Fatalf("rotation failed: %v", err)
	}
	createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-b1", refreshTokenTestTime.Add(time.Hour))

	// When: 계열 A를 로그아웃으로 폐기한 뒤 재사용 사유로 다시 폐기
	revokedAt := refreshTokenTestTime.Add(time.Minute)
	if err := RevokeRefreshTokenFamily(ctx, tx, familyA.FamilyID, models.RefreshTokenRevokedByLogout, revokedAt); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := RevokeRefreshTokenFamily(ctx, tx, familyA.FamilyID, models.RefreshTokenRevokedByReuse, revokedAt.Add(time.Minute)); err != nil {
		t.Fatalf("expected no error on second revoke, got: %v", err)
	}

	// Then: 계열 A의 모든 토큰은 처음 폐기 정보를 유지
	for _, hash := range []string{"hash-a1", "hash-a2"} {
		token, _ := GetRefreshTokenByHash(ctx, tx, []byte(hash))
		if token.RevokedAt == nil || !token.RevokedAt.Equal(revokedAt) {
			t.Errorf("%s RevokedAt = %v, want %v", hash, token.RevokedAt, revokedAt)
		}
		if token.RevokedReason == nil || *token.RevokedReason != models.RefreshTokenRevokedByLogout {
			t.Errorf("%s RevokedReason = %v", hash, token.RevokedReason)
		}
	}

	// Then: 계열 B는 그대로
	other, _ := GetRefreshTokenByHash(ctx, tx, []byte("hash-b1"))
	if other.RevokedAt != nil {
		t.Errorf("expected family B untouched, got RevokedAt %v", other.RevokedAt)
	}
}

// TestDeleteStaleRefreshTokens는 오래된 토큰 정리를 테스트합니다
func TestDeleteStaleRefreshTokens(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	// Given: 기준 시각 7일 전보다 오래 전에 절대 만료된 계열, 폐기된 계열, 유효한 계열
	user := createRefreshTokenTestUser(ctx, t, tx)
	cutoff := refreshTokenTestTime.Add(-7 * 24 * time.Hour)
	createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-expired", cutoff.Add(-time.Hour))
	revoked := createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-revoked-old", refreshTokenTestTime.Add(time.Hour))
	if err := RevokeRefreshTokenFamily(ctx, tx, revoked.FamilyID, models.RefreshTokenRevokedByLogout, cutoff.Add(-time.Hour)); err != nil {
		t.Fatalf("failed to revoke: %v", err)
	}
	createTestRefreshTokenFamily(ctx, t, tx, user.ID, "hash-alive", refreshTokenTestTime.Add(time.Hour))

	// When: 정리
	err := DeleteStaleRefreshTokens(ctx, tx, user.ID, cutoff)

	// Then: 만료·폐기된 계열만 삭제
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for hash, wantExists := range map[string]bool{"hash-expired": false, "hash-revoked-old": false, "hash-alive": true} {
		token, _ := GetRefreshTokenByHash(ctx, tx, []byte(hash))
		if (token != nil) != wantExists {
			t.Errorf("%s exists = %v, want %v", hash, token != nil, wantExists)
		}
	}
}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/database/ -run 'RefreshToken' -count=1`
기대: `undefined: CreateRefreshTokenFamily` 등으로 빌드 실패

- [ ] **Step 3: 구현**

`internal/database/refresh_tokens.go`:

```go
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/june20516/orbithall/internal/models"
)

// refreshTokenColumns는 refresh_tokens 조회 시 공통으로 읽는 컬럼 목록입니다
// scanRefreshToken의 Scan 순서와 같아야 합니다
const refreshTokenColumns = `id, user_id, family_id, parent_id, token_hash, expires_at,
	family_expires_at, used_at, revoked_at, revoked_reason, created_at`

// scanRefreshToken은 refreshTokenColumns 순서의 한 행을 RefreshToken으로 읽습니다
func scanRefreshToken(row *sql.Row) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.FamilyID,
		&token.ParentID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.FamilyExpiresAt,
		&token.UsedAt,
		&token.RevokedAt,
		&token.RevokedReason,
		&token.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// CreateRefreshTokenFamily는 로그인 시 새 계열의 첫 Refresh Token을 저장합니다
// family_id는 DB가 gen_random_uuid()로 만듭니다
func CreateRefreshTokenFamily(ctx context.Context, db DBTX, userID int64, tokenHash []byte, expiresAt, familyExpiresAt time.Time) (*models.RefreshToken, error) {
	query := `
		INSERT INTO refresh_tokens (user_id, family_id, token_hash, expires_at, family_expires_at)
		VALUES ($1, gen_random_uuid(), $2, $3, $4)
		RETURNING ` + refreshTokenColumns

	token, err := scanRefreshToken(db.QueryRowContext(ctx, query, userID, tokenHash, expiresAt, familyExpiresAt))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return token, nil
}

// GetRefreshTokenByHash는 토큰 해시로 Refresh Token을 조회합니다
// 찾지 못한 경우 nil을 반환합니다
func GetRefreshTokenByHash(ctx context.Context, db DBTX, tokenHash []byte) (*models.RefreshToken, error) {
	query := `SELECT ` + refreshTokenColumns + ` FROM refresh_tokens WHERE token_hash = $1`

	token, err := scanRefreshToken(db.QueryRowContext(ctx, query, tokenHash))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token by hash: %w", err)
	}

	return token, nil
}

// GetChildRefreshToken은 parentID 토큰을 회전해 만든 후속 토큰을 조회합니다
// 아직 회전하지 않은 토큰이면 nil을 반환합니다
func GetChildRefreshToken(ctx context.Context, db DBTX, parentID int64) (*models.RefreshToken, error) {
	query := `SELECT ` + refreshTokenColumns + ` FROM refresh_tokens WHERE parent_id = $1`

	token, err := scanRefreshToken(db.QueryRowContext(ctx, query, parentID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get child refresh token: %w", err)
	}

	return token, nil
}

// RotateRefreshToken은 parentID 토큰을 사용 처리하고 같은 계열의 후속 토큰을 저장합니다
//
// 부모가 아직 사용되지 않았고 폐기되지 않은 경우에만 두 작업을 한 SQL 문장으로 처리합니다
// 같은 부모로 동시에 요청이 와도 한 요청만 성공하고, 나머지는 nil을 받습니다
// (부모 행을 먼저 잠근 요청이 커밋되면 다른 요청의 WHERE 조건이 다시 평가되어 0행이 됩니다)
//
// 같은 계열에 폐기된 토큰이 하나라도 있으면 회전하지 않습니다
// 계열 폐기 UPDATE는 문장 시작 시점에 보이는 행만 폐기하므로, 동시에 진행 중이던 회전이 만든 후속 토큰은
// 폐기되지 않은 채 남을 수 있습니다. 이 조건이 그런 토큰으로 세션이 이어지는 것을 막습니다
//
// 부모가 이미 사용되었거나, 폐기되었거나, 계열이 폐기되었으면 nil, nil을 반환합니다
func RotateRefreshToken(ctx context.Context, db DBTX, parentID int64, childHash []byte, childExpiresAt, now time.Time) (*models.RefreshToken, error) {
	// INSERT ... SELECT의 SELECT 목록에 쓴 파라미터는 타입을 추론하지 못하므로 명시적으로 캐스팅합니다
	query := `
		WITH parent AS (
			UPDATE refresh_tokens
			SET used_at = $2
			WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL
				AND NOT EXISTS (
					SELECT 1 FROM refresh_tokens AS revoked
					WHERE revoked.family_id = refresh_tokens.family_id AND revoked.revoked_at IS NOT NULL
				)
			RETURNING id, user_id, family_id, family_expires_at
		)
		INSERT INTO refresh_tokens (user_id, family_id, parent_id, token_hash, expires_at, family_expires_at)
		SELECT user_id, family_id, id, $3::bytea, $4::timestamptz, family_expires_at
		FROM parent
		RETURNING ` + refreshTokenColumns

	token, err := scanRefreshToken(db.QueryRowContext(ctx, query, parentID, now, childHash, childExpiresAt))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to rotate refresh token: %w", err)
	}

	return token, nil
}

// RevokeRefreshTokenFamily는 계열의 모든 토큰을 폐기합니다
// 이미 폐기된 토큰은 처음 폐기 정보를 유지합니다 (여러 번 호출해도 결과가 같음)
func RevokeRefreshTokenFamily(ctx context.Context, db DBTX, familyID string, reason string, now time.Time) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $2, revoked_reason = $3
		WHERE family_id = $1 AND revoked_at IS NULL
	`

	if _, err := db.ExecContext(ctx, query, familyID, now, reason); err != nil {
		return fmt.Errorf("failed to revoke refresh token family: %w", err)
	}

	return nil
}

// DeleteStaleRefreshTokens는 사용자의 토큰 중 before 이전에 절대 만료되었거나 폐기된 행을 삭제합니다
// 로그인 시 호출해 별도 스케줄러 없이 테이블이 계속 커지지 않게 합니다
func DeleteStaleRefreshTokens(ctx context.Context, db DBTX, userID int64, before time.Time) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE user_id = $1 AND (family_expires_at < $2 OR revoked_at < $2)
	`

	if _, err := db.ExecContext(ctx, query, userID, before); err != nil {
		return fmt.Errorf("failed to delete stale refresh tokens: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: 통과 확인**

실행: `go test ./internal/database/ -run 'RefreshToken' -count=1 -v`
기대: 위 테스트 전부 PASS

실행: `go test ./internal/database/ -count=1`
기대: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/database/refresh_tokens.go internal/database/refresh_tokens_test.go
git commit -m "feat: Refresh Token 저장·회전·폐기 쿼리 추가"
```

---

### Task 5: 인증 에러 코드 상수와 JWT 미들웨어 에러 본문 통일

**Files:**
- Modify: `internal/handlers/middleware.go` (에러 코드 `const` 블록, 약 17~41행)
- Modify: `internal/handlers/jwt_middleware.go` (전체)
- Test: `internal/handlers/jwt_middleware_test.go` (끝에 추가)

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/handlers/jwt_middleware_test.go` import에 `"encoding/json"`, `"time"`, `"github.com/golang-jwt/jwt/v5"`를 추가하고, 파일 끝에 다음을 추가한다.

```go
// readErrorCode는 객체 형식 에러 응답 본문({"error":{"code":...}})에서 code를 꺼냅니다
func readErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not in object format: %v, body: %s", err, rec.Body.String())
	}
	return body.Error.Code
}

// TestJWTAuthMiddleware_ErrorBody는 인증 실패 응답이 객체 형식 에러 본문인지 테스트합니다
func TestJWTAuthMiddleware_ErrorBody(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	// 만료된 Access Token
	expiredClaims := &auth.CustomClaims{
		UserID:    1,
		Email:     "expired@example.com",
		TokenType: auth.AccessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    auth.TokenIssuer,
			Audience:  jwt.ClaimStrings{auth.AdminAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	tests := []struct {
		name          string
		authorization string
		wantCode      string
	}{
		{name: "헤더 없음", authorization: "", wantCode: ErrMissingToken},
		{name: "Bearer 형식 아님", authorization: "Token abc", wantCode: ErrInvalidToken},
		{name: "만료된 토큰", authorization: "Bearer " + expiredToken, wantCode: ErrExpiredToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 인증에 실패하는 요청
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("next handler should not be called")
			})
			req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()

			// When: 미들웨어 실행
			JWTAuthMiddleware(db)(nextHandler).ServeHTTP(rec, req)

			// Then: 401과 객체 형식 에러 코드
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if code := readErrorCode(t, rec); code != tt.wantCode {
				t.Errorf("error.code = %q, want %q", code, tt.wantCode)
			}
		})
	}
}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/handlers/ -run TestJWTAuthMiddleware_ErrorBody -count=1`
기대: `undefined: ErrMissingToken` 등으로 빌드 실패

- [ ] **Step 3: 에러 코드 상수 추가**

`internal/handlers/middleware.go`의 에러 코드 `const` 블록에서 `// Rate limiting 에러` 주석 바로 위에 추가한다.

```go
	// 어드민 인증 에러
	ErrMissingToken        = "MISSING_TOKEN"         // Authorization 헤더 없음
	ErrInvalidToken        = "INVALID_TOKEN"         // Access Token 형식 오류 또는 서명 불일치
	ErrExpiredToken        = "EXPIRED_TOKEN"         // Access Token 만료
	ErrUserNotFound        = "USER_NOT_FOUND"        // 토큰의 사용자가 존재하지 않음
	ErrInvalidIDToken      = "INVALID_ID_TOKEN"      // Google ID Token 검증 실패
	ErrInvalidRefreshToken = "INVALID_REFRESH_TOKEN" // Refresh Token이 없거나 폐기됨
	ErrRefreshTokenExpired = "REFRESH_TOKEN_EXPIRED" // Refresh Token 만료
	ErrRefreshTokenReused  = "REFRESH_TOKEN_REUSED"  // 사용된 Refresh Token 재사용 (세션 폐기됨)

```

- [ ] **Step 4: JWT 미들웨어 수정**

`internal/handlers/jwt_middleware.go` 전체를 다음으로 바꾼다. 에러 응답은 `respondError`(middleware.go)를 쓰고, 파일 전용 헬퍼 `respondWithError`와 `encoding/json` import는 삭제한다.

```go
package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
)

// userContextKey는 Context에 사용자 정보를 저장할 때 사용하는 키입니다
const userContextKey contextKey = "user"

// JWTAuthMiddleware는 JWT 기반 인증 미들웨어입니다
// Authorization 헤더에서 Bearer 토큰을 추출하고 검증합니다
// 실패 시 {"error":{"code","message"}} 형식으로 응답합니다
func JWTAuthMiddleware(db database.DBTX) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Authorization 헤더 추출
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, http.StatusUnauthorized, ErrMissingToken, "Authorization header is required", nil)
				return
			}

			// 2. Bearer 형식 검증
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondError(w, http.StatusUnauthorized, ErrInvalidToken, "Authorization header must be in format: Bearer {token}", nil)
				return
			}

			tokenString := parts[1]

			// 3. JWT 토큰 검증
			claims, err := auth.ValidateJWT(tokenString)
			if err != nil {
				if err == auth.ErrExpiredToken {
					respondError(w, http.StatusUnauthorized, ErrExpiredToken, "Token has expired", nil)
					return
				}
				respondError(w, http.StatusUnauthorized, ErrInvalidToken, "Invalid token", nil)
				return
			}

			// 4. 사용자 조회
			user, err := database.GetUserByID(r.Context(), db, claims.UserID)
			if err != nil {
				respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get user", nil)
				return
			}

			// 5. 사용자 존재 여부 확인
			if user == nil {
				respondError(w, http.StatusUnauthorized, ErrUserNotFound, "User not found", nil)
				return
			}

			// 6. Context에 사용자 정보 저장
			ctx := SetUserInContext(r.Context(), user)

			// 7. 다음 핸들러 호출
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SetUserInContext는 Context에 사용자 정보를 저장합니다
func SetUserInContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUserFromContext는 Context에서 사용자 정보를 추출합니다
// 사용자 정보가 없으면 nil을 반환합니다
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(userContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
```

- [ ] **Step 5: 통과 확인**

실행: `grep -rn "respondWithError" internal/`
기대: 출력 없음

실행: `go test ./internal/handlers/ -run TestJWTAuthMiddleware -count=1`
기대: `ok`

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/middleware.go internal/handlers/jwt_middleware.go internal/handlers/jwt_middleware_test.go
git commit -m "refactor: JWT 미들웨어 에러 본문을 객체 형식으로 통일"
```

---

### Task 6: 세션 규칙 (`internal/handlers/session.go`)

발급·회전·유예 시간·재사용 탐지·폐기 규칙을 HTTP와 분리해 구현한다. 모든 함수는 `database.DBTX`와 `now`를 받아 테스트 트랜잭션 안에서 시간을 고정해 검증한다.

**Files:**
- Create: `internal/handlers/session.go`
- Test: `internal/handlers/session_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/handlers/session_test.go`:

```go
package handlers

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/ratelimit"
	"github.com/june20516/orbithall/internal/testhelpers"
	"golang.org/x/time/rate"
)

func init() {
	// 테스트용 Refresh Token 비밀키 (JWT_SECRET과 다른 값)
	os.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-secret-for-handlers-at-least-32-chars")
}

// sessionTestTime은 세션 테스트의 기준 시각입니다
var sessionTestTime = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// sessionTestConfig는 명세 기본값과 같은 수명 설정입니다
func sessionTestConfig() auth.RefreshTokenConfig {
	return auth.RefreshTokenConfig{
		IdleTTL:     14 * 24 * time.Hour,
		AbsoluteTTL: 30 * 24 * time.Hour,
		ReuseGrace:  30 * time.Second,
	}
}

// unlimitedRefreshLimiter는 요청 제한이 없는 limiter입니다
func unlimitedRefreshLimiter() *ratelimit.RateLimiter {
	return ratelimit.NewRateLimiter(rate.Inf, 1)
}

// createSessionTestUser는 세션 테스트용 사용자를 만듭니다
func createSessionTestUser(ctx context.Context, t *testing.T, tx database.DBTX) *models.User {
	t.Helper()
	user := &models.User{
		Email:    "session@example.com",
		Name:     "Session User",
		GoogleID: "google-session-user",
	}
	if err := database.CreateUser(ctx, tx, user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// mustIssueSession은 기준 시각에 로그인 세션을 발급합니다
func mustIssueSession(ctx context.Context, t *testing.T, tx database.DBTX, user *models.User, cfg auth.RefreshTokenConfig) *TokenPairResponse {
	t.Helper()
	pair, err := issueSession(ctx, tx, user, cfg, sessionTestTime)
	if err != nil {
		t.Fatalf("failed to issue session: %v", err)
	}
	return pair
}

// mustRotate는 회전이 성공해야 하는 경우에 사용합니다
func mustRotate(ctx context.Context, t *testing.T, tx database.DBTX, refreshToken string, cfg auth.RefreshTokenConfig, now time.Time) *TokenPairResponse {
	t.Helper()
	pair, err := rotateSession(ctx, tx, refreshToken, cfg, unlimitedRefreshLimiter(), now)
	if err != nil {
		t.Fatalf("expected rotation to succeed, got: %v", err)
	}
	return pair
}

// TestIssueSession은 로그인 시 세션 발급을 테스트합니다
func TestIssueSession(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("토큰 쌍을 발급하고 Refresh Token 해시를 저장한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자
		user := createSessionTestUser(ctx, t, tx)

		// When: 세션 발급
		pair, err := issueSession(ctx, tx, user, sessionTestConfig(), sessionTestTime)

		// Then: 토큰 쌍 필드가 채워짐
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if pair.TokenType != "Bearer" || pair.AccessToken == "" || pair.AccessTokenExpiresAt.IsZero() {
			t.Errorf("unexpected access token fields: %+v", pair)
		}
		if !strings.HasPrefix(pair.RefreshToken, auth.RefreshTokenPrefix) {
			t.Errorf("refresh token has no prefix: %q", pair.RefreshToken)
		}
		wantRefreshExpiresAt := sessionTestTime.Add(14 * 24 * time.Hour)
		if !pair.RefreshTokenExpiresAt.Equal(wantRefreshExpiresAt) {
			t.Errorf("RefreshTokenExpiresAt = %v, want %v", pair.RefreshTokenExpiresAt, wantRefreshExpiresAt)
		}

		// Then: 원문이 아닌 해시로 저장되고 절대 만료는 30일 뒤
		stored, err := database.GetRefreshTokenByHash(ctx, tx, auth.HashRefreshToken(pair.RefreshToken))
		if err != nil || stored == nil {
			t.Fatalf("expected stored token, got %+v, %v", stored, err)
		}
		if stored.UserID != user.ID {
			t.Errorf("UserID = %d, want %d", stored.UserID, user.ID)
		}
		if !stored.FamilyExpiresAt.Equal(sessionTestTime.Add(30 * 24 * time.Hour)) {
			t.Errorf("FamilyExpiresAt = %v", stored.FamilyExpiresAt)
		}
	})

	t.Run("오래된 토큰을 정리한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 8일 전에 절대 만료된 계열
		user := createSessionTestUser(ctx, t, tx)
		staleExpiresAt := sessionTestTime.Add(-8 * 24 * time.Hour)
		if _, err := database.CreateRefreshTokenFamily(ctx, tx, user.ID, []byte("stale-hash"), staleExpiresAt, staleExpiresAt); err != nil {
			t.Fatalf("failed to create stale token: %v", err)
		}

		// When: 새 로그인
		mustIssueSession(ctx, t, tx, user, sessionTestConfig())

		// Then: 오래된 계열이 삭제됨
		stale, _ := database.GetRefreshTokenByHash(ctx, tx, []byte("stale-hash"))
		if stale != nil {
			t.Error("expected stale token to be deleted")
		}
	})
}

// TestRotateSession은 Refresh Token 회전 규칙을 테스트합니다
func TestRotateSession(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	cfg := sessionTestConfig()

	t.Run("새 토큰 쌍을 발급하고 이전 토큰을 사용 처리한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)

		// When: 1분 뒤 회전
		now := sessionTestTime.Add(time.Minute)
		pair, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), now)

		// Then: 새 Refresh Token과 연장된 만료
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if pair.RefreshToken == login.RefreshToken {
			t.Error("expected a new refresh token")
		}
		if pair.AccessToken == "" {
			t.Error("expected an access token")
		}
		if !pair.RefreshTokenExpiresAt.Equal(now.Add(cfg.IdleTTL)) {
			t.Errorf("RefreshTokenExpiresAt = %v, want %v", pair.RefreshTokenExpiresAt, now.Add(cfg.IdleTTL))
		}

		// Then: 이전 토큰은 사용 처리됨
		old, _ := database.GetRefreshTokenByHash(ctx, tx, auth.HashRefreshToken(login.RefreshToken))
		if old.UsedAt == nil {
			t.Error("expected previous token to be marked used")
		}
	})

	t.Run("회전한 토큰으로 다시 회전할 수 있다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 한 번 회전한 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, sessionTestTime.Add(time.Hour))

		// When: 하루 뒤 두 번째 토큰으로 회전
		third, err := rotateSession(ctx, tx, second.RefreshToken, cfg, unlimitedRefreshLimiter(), sessionTestTime.Add(25*time.Hour))

		// Then: 성공하고 또 다른 토큰
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if third.RefreshToken == second.RefreshToken || third.RefreshToken == login.RefreshToken {
			t.Error("expected a new refresh token")
		}
	})

	t.Run("유예 시간 안에 이전 토큰을 다시 내면 같은 후속 토큰을 돌려준다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		first := mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)

		// When: 10초 뒤 R1을 다시 제출
		again, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(10*time.Second))

		// Then: 같은 R2와 같은 만료 시각
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if again.RefreshToken != first.RefreshToken {
			t.Errorf("expected the same R2, got a different token")
		}
		if !again.RefreshTokenExpiresAt.Equal(first.RefreshTokenExpiresAt) {
			t.Errorf("RefreshTokenExpiresAt = %v, want %v", again.RefreshTokenExpiresAt, first.RefreshTokenExpiresAt)
		}
		if again.AccessToken == "" {
			t.Error("expected an access token")
		}
	})

	t.Run("유예 시간이 지나 이전 토큰을 다시 내면 계열 전체를 폐기한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)

		// When: 31초 뒤 R1을 다시 제출
		_, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(31*time.Second))

		// Then: 재사용 에러
		if !errors.Is(err, errSessionReused) {
			t.Fatalf("expected errSessionReused, got: %v", err)
		}

		// Then: R2도 폐기되어 쓸 수 없음
		_, err = rotateSession(ctx, tx, second.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(time.Minute))
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid for R2, got: %v", err)
		}
	})

	t.Run("후속 토큰이 이미 사용되었으면 유예 시간 안이라도 재사용으로 본다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 -> R3 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)
		mustRotate(ctx, t, tx, second.RefreshToken, cfg, rotatedAt.Add(5*time.Second))

		// When: R1을 10초 뒤 다시 제출
		_, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(10*time.Second))

		// Then: 재사용 에러
		if !errors.Is(err, errSessionReused) {
			t.Errorf("expected errSessionReused, got: %v", err)
		}
	})

	t.Run("유휴 만료 시각이 되면 만료 에러", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)

		// When: 14일 뒤 회전
		_, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), sessionTestTime.Add(cfg.IdleTTL))

		// Then: 만료 에러
		if !errors.Is(err, errSessionExpired) {
			t.Errorf("expected errSessionExpired, got: %v", err)
		}
	})

	t.Run("절대 만료를 넘겨 연장되지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 유휴 30분, 절대 1시간 설정으로 로그인
		shortCfg := auth.RefreshTokenConfig{IdleTTL: 30 * time.Minute, AbsoluteTTL: time.Hour, ReuseGrace: 30 * time.Second}
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, shortCfg)

		// When: 20분, 45분에 회전
		second := mustRotate(ctx, t, tx, login.RefreshToken, shortCfg, sessionTestTime.Add(20*time.Minute))
		third := mustRotate(ctx, t, tx, second.RefreshToken, shortCfg, sessionTestTime.Add(45*time.Minute))

		// Then: 세 번째 토큰의 만료는 절대 만료(로그인 + 1시간)로 제한됨
		absoluteExpiresAt := sessionTestTime.Add(time.Hour)
		if !third.RefreshTokenExpiresAt.Equal(absoluteExpiresAt) {
			t.Errorf("RefreshTokenExpiresAt = %v, want %v", third.RefreshTokenExpiresAt, absoluteExpiresAt)
		}

		// Then: 절대 만료 시각에는 만료 에러
		_, err := rotateSession(ctx, tx, third.RefreshToken, shortCfg, unlimitedRefreshLimiter(), absoluteExpiresAt)
		if !errors.Is(err, errSessionExpired) {
			t.Errorf("expected errSessionExpired, got: %v", err)
		}
	})

	t.Run("모르는 토큰은 무효", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 저장되지 않은 토큰으로 회전
		_, err := rotateSession(ctx, tx, "ohrt_unknown", cfg, unlimitedRefreshLimiter(), sessionTestTime)

		// Then: 무효 에러
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid, got: %v", err)
		}
	})

	t.Run("계열당 요청이 너무 많으면 제한한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 한 시간에 한 번만 허용하는 limiter와 로그인 세션
		limiter := ratelimit.NewRateLimiter(rate.Every(time.Hour), 1)
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		second, err := rotateSession(ctx, tx, login.RefreshToken, cfg, limiter, sessionTestTime.Add(time.Minute))
		if err != nil {
			t.Fatalf("first rotation failed: %v", err)
		}

		// When: 같은 계열로 바로 다시 회전
		_, err = rotateSession(ctx, tx, second.RefreshToken, cfg, limiter, sessionTestTime.Add(2*time.Minute))

		// Then: 제한 에러
		if !errors.Is(err, errSessionRateLimited) {
			t.Errorf("expected errSessionRateLimited, got: %v", err)
		}
	})
}

// TestRevokeSession은 로그아웃을 테스트합니다
func TestRevokeSession(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	cfg := sessionTestConfig()

	t.Run("계열을 폐기해 이후 회전을 막는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)

		// When: 로그아웃
		err := revokeSession(ctx, tx, login.RefreshToken, sessionTestTime.Add(time.Minute))

		// Then: 에러 없고 이후 회전은 무효
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		_, err = rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), sessionTestTime.Add(2*time.Minute))
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid, got: %v", err)
		}
	})

	t.Run("모르는 토큰이어도 에러 없이 끝난다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 저장되지 않은 토큰으로 로그아웃
		err := revokeSession(ctx, tx, "ohrt_unknown", sessionTestTime)

		// Then: 에러 없음
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/handlers/ -run 'TestIssueSession|TestRotateSession|TestRevokeSession' -count=1`
기대: `undefined: issueSession` 등으로 빌드 실패

- [ ] **Step 3: 구현**

`internal/handlers/session.go`:

```go
package handlers

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/ratelimit"
)

// staleRefreshTokenRetention은 절대 만료되거나 폐기된 Refresh Token을 삭제하기 전까지 보관하는 기간입니다
// 폐기 직후의 재사용 시도를 조사할 수 있도록 바로 지우지 않습니다
const staleRefreshTokenRetention = 7 * 24 * time.Hour

// 세션 규칙 위반 에러
// SessionHandler가 각 에러를 HTTP 상태와 에러 코드로 바꿉니다
var (
	errSessionInvalid     = errors.New("refresh token is invalid or revoked")
	errSessionExpired     = errors.New("refresh token has expired")
	errSessionReused      = errors.New("refresh token was reused")
	errSessionUserMissing = errors.New("session user not found")
	errSessionRateLimited = errors.New("too many refresh requests")
)

// TokenPairResponse는 로그인과 토큰 갱신 응답에 공통으로 들어가는 토큰 정보입니다
// @Description Access Token과 Refresh Token 발급 결과 (시각은 RFC 3339, UTC)
type TokenPairResponse struct {
	TokenType             string    `json:"token_type" example:"Bearer"`
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token" example:"ohrt_..."`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

// issueSession은 로그인한 사용자에게 새 세션(Access Token + 새 계열의 첫 Refresh Token)을 발급합니다
// 같은 사용자의 오래된 Refresh Token도 함께 정리합니다
func issueSession(ctx context.Context, db database.DBTX, user *models.User, cfg auth.RefreshTokenConfig, now time.Time) (*TokenPairResponse, error) {
	if err := database.DeleteStaleRefreshTokens(ctx, db, user.ID, now.Add(-staleRefreshTokenRetention)); err != nil {
		return nil, err
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	familyExpiresAt := now.Add(cfg.AbsoluteTTL)
	stored, err := database.CreateRefreshTokenFamily(
		ctx, db, user.ID,
		auth.HashRefreshToken(refreshToken),
		refreshTokenExpiry(now, cfg.IdleTTL, familyExpiresAt),
		familyExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	return newTokenPairResponse(user, refreshToken, stored.ExpiresAt)
}

// rotateSession은 제출된 Refresh Token을 한 번 사용 처리하고 새 토큰 쌍을 발급합니다
// 규칙은 docs/specs/admin-auth-token-refresh.md 3장을 따릅니다
func rotateSession(ctx context.Context, db database.DBTX, presented string, cfg auth.RefreshTokenConfig, limiter *ratelimit.RateLimiter, now time.Time) (*TokenPairResponse, error) {
	presentedHash := auth.HashRefreshToken(presented)

	current, err := database.GetRefreshTokenByHash(ctx, db, presentedHash)
	if err != nil {
		return nil, err
	}
	if current == nil || current.RevokedAt != nil {
		return nil, errSessionInvalid
	}

	// 계열(로그인 세션) 단위로 갱신 빈도를 제한합니다
	// 모든 요청이 어드민 서버 IP 하나에서 오므로 IP 기준 제한은 쓰지 않습니다
	if !limiter.GetLimiter(current.FamilyID).Allow() {
		return nil, errSessionRateLimited
	}

	if current.UsedAt != nil {
		return reissueWithinGrace(ctx, db, current, presented, cfg, now)
	}
	if current.IsExpired(now) {
		return nil, errSessionExpired
	}

	nextToken, err := auth.DeriveNextRefreshToken(presented)
	if err != nil {
		return nil, err
	}

	next, err := database.RotateRefreshToken(
		ctx, db, current.ID,
		auth.HashRefreshToken(nextToken),
		refreshTokenExpiry(now, cfg.IdleTTL, current.FamilyExpiresAt),
		now,
	)
	if err != nil {
		return nil, err
	}

	if next == nil {
		// 같은 토큰으로 들어온 다른 요청이 먼저 회전했거나, 계열이 폐기되었습니다
		// 상태를 다시 읽어, 사용 처리된 경우에만 유예 시간 규칙으로 판단합니다
		current, err = database.GetRefreshTokenByHash(ctx, db, presentedHash)
		if err != nil {
			return nil, err
		}
		if current == nil || current.RevokedAt != nil || current.UsedAt == nil {
			return nil, errSessionInvalid
		}
		return reissueWithinGrace(ctx, db, current, presented, cfg, now)
	}

	return buildSessionResponse(ctx, db, next.UserID, nextToken, next.ExpiresAt)
}

// reissueWithinGrace는 이미 사용된 토큰이 다시 제출되었을 때를 처리합니다
//
// 유예 시간 안이고 후속 토큰이 아직 쓰이지 않았다면 그 후속 토큰을 다시 돌려줍니다
// 새 토큰을 만들지 않으므로 계열이 한 줄로 유지되고, 탈취된 경우에도 나중에 후속 토큰을 쓰는 쪽에서 재사용이 탐지됩니다
// 그 외에는 탈취로 보고 계열 전체를 폐기합니다
func reissueWithinGrace(ctx context.Context, db database.DBTX, used *models.RefreshToken, presented string, cfg auth.RefreshTokenConfig, now time.Time) (*TokenPairResponse, error) {
	if now.Sub(*used.UsedAt) <= cfg.ReuseGrace {
		child, err := database.GetChildRefreshToken(ctx, db, used.ID)
		if err != nil {
			return nil, err
		}

		if child != nil && child.UsedAt == nil && child.RevokedAt == nil && !child.IsExpired(now) {
			// 후속 토큰은 이전 토큰에서 결정적으로 만들어지므로 원문을 저장하지 않고도 다시 계산할 수 있습니다
			childToken, err := auth.DeriveNextRefreshToken(presented)
			if err != nil {
				return nil, err
			}

			// 비밀키가 바뀐 뒤라면 계산한 값이 저장된 해시와 달라 돌려줄 수 없습니다
			if bytes.Equal(auth.HashRefreshToken(childToken), child.TokenHash) {
				return buildSessionResponse(ctx, db, child.UserID, childToken, child.ExpiresAt)
			}
		}
	}

	if err := database.RevokeRefreshTokenFamily(ctx, db, used.FamilyID, models.RefreshTokenRevokedByReuse, now); err != nil {
		return nil, err
	}

	return nil, errSessionReused
}

// revokeSession은 제출된 Refresh Token이 속한 계열을 폐기합니다 (로그아웃)
// 토큰이 없거나 이미 폐기된 경우에도 에러 없이 끝납니다
func revokeSession(ctx context.Context, db database.DBTX, presented string, now time.Time) error {
	current, err := database.GetRefreshTokenByHash(ctx, db, auth.HashRefreshToken(presented))
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}

	return database.RevokeRefreshTokenFamily(ctx, db, current.FamilyID, models.RefreshTokenRevokedByLogout, now)
}

// buildSessionResponse는 토큰 소유자를 조회해 새 Access Token과 함께 응답을 만듭니다
func buildSessionResponse(ctx context.Context, db database.DBTX, userID int64, refreshToken string, refreshExpiresAt time.Time) (*TokenPairResponse, error) {
	user, err := database.GetUserByID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errSessionUserMissing
	}

	return newTokenPairResponse(user, refreshToken, refreshExpiresAt)
}

// newTokenPairResponse는 Access Token을 새로 발급해 토큰 쌍 응답을 만듭니다
// 시각은 UTC 초 단위로 맞춥니다 (DB 값의 마이크로초는 버리므로 실제 만료보다 최대 1초 이르게 표시됩니다)
func newTokenPairResponse(user *models.User, refreshToken string, refreshExpiresAt time.Time) (*TokenPairResponse, error) {
	accessToken, accessExpiresAt, err := auth.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &TokenPairResponse{
		TokenType:             "Bearer",
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt.UTC(),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiresAt.UTC().Truncate(time.Second),
	}, nil
}

// refreshTokenExpiry는 새 Refresh Token의 만료 시각을 계산합니다
// 유휴 만료(now + idleTTL)가 계열의 절대 만료보다 늦으면 절대 만료를 씁니다
func refreshTokenExpiry(now time.Time, idleTTL time.Duration, familyExpiresAt time.Time) time.Time {
	idleExpiresAt := now.Add(idleTTL)
	if idleExpiresAt.After(familyExpiresAt) {
		return familyExpiresAt
	}
	return idleExpiresAt
}
```

- [ ] **Step 4: 통과 확인**

실행: `go test ./internal/handlers/ -run 'TestIssueSession|TestRotateSession|TestRevokeSession' -count=1 -v`
기대: 모든 서브테스트 PASS

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/session.go internal/handlers/session_test.go
git commit -m "feat: Refresh Token 세션 발급·회전·폐기 규칙 추가"
```

---

### Task 7: 로그인 응답에 토큰 쌍 추가와 에러 본문 통일 (`auth.go`)

`GoogleVerify`의 성공 경로는 실제 Google ID Token이 필요해 단위 테스트할 수 없다(기존 `auth_test.go` 주석과 같음). 세션 발급 자체는 Task 6의 `TestIssueSession`이 검증하고, 여기서는 에러 본문 형식을 테스트한다.

**Files:**
- Modify: `internal/handlers/auth.go`
- Test: `internal/handlers/auth_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/handlers/auth_test.go`의 기존 테스트 네 개에 에러 코드 검증을 추가한다. 각 테스트의 `// Then:` 상태 코드 검사 바로 아래에 넣는다.

`TestGoogleVerify_MissingFields` (루프 안), `TestGoogleVerify_InvalidContentType`, `TestGoogleVerify_InvalidJSONBody`:

```go
			if code := readErrorCode(t, rec); code != ErrInvalidInput {
				t.Errorf("error.code = %q, want %q", code, ErrInvalidInput)
			}
```

(`TestGoogleVerify_InvalidContentType`, `TestGoogleVerify_InvalidJSONBody`는 루프 밖이므로 들여쓰기를 한 단계 줄인다.)

`TestGoogleVerify_InvalidGoogleToken`:

```go
	if code := readErrorCode(t, rec); code != ErrInvalidIDToken {
		t.Errorf("error.code = %q, want %q", code, ErrInvalidIDToken)
	}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/handlers/ -run TestGoogleVerify -count=1`
기대: `error body is not in object format`으로 FAIL (현재는 평문 응답)

- [ ] **Step 3: 구현**

`internal/handlers/auth.go`를 다음처럼 바꾼다.

import 블록:

```go
import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
)
```

`AuthHandler`와 생성자:

```go
// AuthHandler는 인증 관련 HTTP 요청을 처리합니다
type AuthHandler struct {
	db            *sql.DB
	refreshConfig auth.RefreshTokenConfig
	// now는 현재 시각을 반환합니다 (테스트에서 시간을 고정하기 위해 교체할 수 있음)
	now func() time.Time
}

// NewAuthHandler는 AuthHandler의 새 인스턴스를 생성합니다
func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{
		db:            db,
		refreshConfig: auth.LoadRefreshTokenConfig(),
		now:           time.Now,
	}
}
```

`GoogleVerifyResponse`:

```go
// GoogleVerifyResponse는 Google ID Token 검증 성공 응답입니다
type GoogleVerifyResponse struct {
	TokenPairResponse

	// Token은 access_token과 같은 값입니다 (token 필드만 읽는 클라이언트 호환용, 전환 3단계에서 제거)
	Token string `json:"token"`

	User *models.User `json:"user"`
}
```

`GoogleVerify` 함수 전체(swag 주석 포함)를 다음으로 바꾼다.

```go
// GoogleVerify는 Google ID Token을 검증하고 Access Token과 Refresh Token을 발급합니다
//
// @Summary      Google OAuth 인증 및 토큰 발급
// @Description  Google ID Token을 검증하고 사용자를 생성/조회한 후 Access Token과 Refresh Token을 발급합니다
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body GoogleVerifyRequest true "Google 인증 정보"
// @Success      200 {object} GoogleVerifyResponse "토큰 쌍 및 사용자 정보"
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "INVALID_ID_TOKEN"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Router       /auth/google/verify [post]
func (h *AuthHandler) GoogleVerify(w http.ResponseWriter, r *http.Request) {
	// 1. Content-Type 검증
	if r.Header.Get("Content-Type") != "application/json" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Content-Type must be application/json", nil)
		return
	}

	// 2. JSON 요청 파싱
	var req GoogleVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid JSON", nil)
		return
	}

	// 3. 입력 검증
	if req.IDToken == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "id_token is required", nil)
		return
	}
	if req.Email == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "email is required", nil)
		return
	}
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "name is required", nil)
		return
	}

	// 4. Google ID Token 검증
	payload, err := auth.VerifyGoogleIDToken(r.Context(), req.IDToken)
	if err != nil {
		if err == auth.ErrInvalidIDToken {
			respondError(w, http.StatusUnauthorized, ErrInvalidIDToken, "Invalid Google ID Token", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to verify Google ID Token", nil)
		return
	}

	// 5. 트랜잭션 시작
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to start transaction", nil)
		return
	}
	defer tx.Rollback()

	// 6. Google ID로 사용자 조회
	user, err := database.GetUserByGoogleID(r.Context(), tx, payload.GoogleID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get user", nil)
		return
	}

	// 7. 사용자가 없으면 생성
	if user == nil {
		user = &models.User{
			Email:      req.Email,
			Name:       req.Name,
			PictureURL: req.Picture,
			GoogleID:   payload.GoogleID,
		}

		if err := database.CreateUser(r.Context(), tx, user); err != nil {
			respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to create user", nil)
			return
		}
	}

	// 8. 세션 발급 (Access Token + Refresh Token)
	// Refresh Token 저장도 사용자 생성과 같은 트랜잭션에서 처리합니다
	pair, err := issueSession(r.Context(), tx, user, h.refreshConfig, h.now())
	if err != nil {
		log.Printf("[ERROR] failed to issue session: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to issue session", nil)
		return
	}

	// 9. 트랜잭션 커밋
	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to commit transaction", nil)
		return
	}

	// 10. 응답
	respondJSON(w, http.StatusOK, GoogleVerifyResponse{
		TokenPairResponse: *pair,
		Token:             pair.AccessToken,
		User:              user,
	})
}
```

- [ ] **Step 4: 통과 확인**

실행: `go test ./internal/handlers/ -run TestGoogleVerify -count=1`
기대: `ok`

실행: `grep -n "http.Error" internal/handlers/auth.go`
기대: 출력 없음

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/auth.go internal/handlers/auth_test.go
git commit -m "feat: 로그인 시 Refresh Token 발급, 인증 에러 본문 객체 형식 적용"
```

---

### Task 8: `/auth/refresh`, `/auth/logout` 핸들러

**Files:**
- Create: `internal/handlers/session_handler.go`
- Test: `internal/handlers/session_handler_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/handlers/session_handler_test.go`:

```go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/ratelimit"
	"github.com/june20516/orbithall/internal/testhelpers"
	"golang.org/x/time/rate"
)

// newTestSessionHandler는 현재 시각을 now로 고정한 SessionHandler를 만듭니다
func newTestSessionHandler(tx database.DBTX, now time.Time) *SessionHandler {
	handler := NewSessionHandler(tx)
	handler.now = func() time.Time { return now }
	return handler
}

// postSessionRequest는 refresh_token을 담은 JSON 요청을 handler에 보냅니다
func postSessionRequest(handler http.HandlerFunc, path string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

// refreshTokenBody는 {"refresh_token": token} JSON 문자열을 만듭니다
func refreshTokenBody(t *testing.T, token string) string {
	t.Helper()
	body, err := json.Marshal(RefreshTokenRequest{RefreshToken: token})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	return string(body)
}

// TestSessionHandler_Refresh는 POST /auth/refresh를 테스트합니다
func TestSessionHandler_Refresh(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("성공하면 200과 새 토큰 쌍", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		handler := newTestSessionHandler(tx, sessionTestTime.Add(time.Minute))

		// When: 갱신 요청
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))

		// Then: 200과 토큰 쌍
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
		}
		var pair TokenPairResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &pair); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if pair.TokenType != "Bearer" || pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Errorf("unexpected response: %+v", pair)
		}
		if pair.RefreshToken == login.RefreshToken {
			t.Error("expected a new refresh token")
		}
	})

	t.Run("refresh_token이 없으면 400 INVALID_INPUT", func(t *testing.T) {
		_, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 빈 본문과 잘못된 JSON
		handler := newTestSessionHandler(tx, sessionTestTime)
		for _, body := range []string{`{}`, `{invalid`} {
			rec := postSessionRequest(handler.Refresh, "/auth/refresh", body)

			// Then: 400
			if rec.Code != http.StatusBadRequest {
				t.Errorf("body %s: status = %d", body, rec.Code)
			}
			if code := readErrorCode(t, rec); code != ErrInvalidInput {
				t.Errorf("body %s: error.code = %q", body, code)
			}
		}
	})

	t.Run("모르는 토큰은 401 INVALID_REFRESH_TOKEN", func(t *testing.T) {
		_, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 저장되지 않은 토큰으로 갱신
		handler := newTestSessionHandler(tx, sessionTestTime)
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, "ohrt_unknown"))

		// Then: 401
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrInvalidRefreshToken {
			t.Errorf("error.code = %q", code)
		}
	})

	t.Run("만료된 토큰은 401 REFRESH_TOKEN_EXPIRED", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션, 15일 뒤
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		handler := newTestSessionHandler(tx, sessionTestTime.Add(15*24*time.Hour))

		// When: 갱신
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))

		// Then: 401
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrRefreshTokenExpired {
			t.Errorf("error.code = %q", code)
		}
	})

	t.Run("유예 시간 뒤 재사용은 401 REFRESH_TOKEN_REUSED", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 한 번 회전한 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		mustRotate(ctx, t, tx, login.RefreshToken, sessionTestConfig(), sessionTestTime)

		// When: 1분 뒤 이전 토큰으로 갱신
		handler := newTestSessionHandler(tx, sessionTestTime.Add(time.Minute))
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))

		// Then: 401
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrRefreshTokenReused {
			t.Errorf("error.code = %q", code)
		}
	})

	t.Run("요청 제한을 넘으면 429와 Retry-After", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 요청을 모두 거부하는 limiter
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		handler := newTestSessionHandler(tx, sessionTestTime)
		handler.limiter = ratelimit.NewRateLimiter(rate.Every(time.Hour), 0)

		// When: 갱신
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))

		// Then: 429
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrRateLimitExceeded {
			t.Errorf("error.code = %q", code)
		}
		if rec.Header().Get("Retry-After") == "" {
			t.Error("expected Retry-After header")
		}
	})
}

// TestSessionHandler_Logout은 POST /auth/logout을 테스트합니다
func TestSessionHandler_Logout(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("204를 반환하고 이후 갱신을 막는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		handler := newTestSessionHandler(tx, sessionTestTime.Add(time.Minute))

		// When: 로그아웃
		rec := postSessionRequest(handler.Logout, "/auth/logout", refreshTokenBody(t, login.RefreshToken))

		// Then: 204, 본문 없음
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, body: %s", rec.Code, rec.Body.String())
		}
		if rec.Body.Len() != 0 {
			t.Errorf("expected empty body, got %s", rec.Body.String())
		}

		// Then: 같은 토큰으로 갱신하면 401 INVALID_REFRESH_TOKEN
		refreshRec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))
		if code := readErrorCode(t, refreshRec); code != ErrInvalidRefreshToken {
			t.Errorf("error.code after logout = %q", code)
		}
	})

	t.Run("모르는 토큰이어도 204", func(t *testing.T) {
		_, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 저장되지 않은 토큰으로 로그아웃
		handler := newTestSessionHandler(tx, sessionTestTime)
		rec := postSessionRequest(handler.Logout, "/auth/logout", refreshTokenBody(t, "ohrt_unknown"))

		// Then: 204
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d", rec.Code)
		}
	})

	t.Run("refresh_token이 없으면 400 INVALID_INPUT", func(t *testing.T) {
		_, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 빈 본문으로 로그아웃
		handler := newTestSessionHandler(tx, sessionTestTime)
		rec := postSessionRequest(handler.Logout, "/auth/logout", `{}`)

		// Then: 400
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrInvalidInput {
			t.Errorf("error.code = %q", code)
		}
	})
}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./internal/handlers/ -run TestSessionHandler -count=1`
기대: `undefined: NewSessionHandler` 등으로 빌드 실패

- [ ] **Step 3: 구현**

`internal/handlers/session_handler.go`:

```go
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/ratelimit"
	"golang.org/x/time/rate"
)

// maxSessionRequestBytes는 토큰 갱신·로그아웃 요청 본문의 최대 크기입니다
const maxSessionRequestBytes = 4 << 10 // 4KB

// 토큰 갱신 요청 제한: 계열(로그인 세션)당 분당 10회
const (
	refreshRateInterval = time.Minute / 10
	refreshRateBurst    = 10
)

// RefreshTokenRequest는 토큰 갱신과 로그아웃 요청 본문입니다
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"ohrt_..."`
}

// SessionHandler는 Refresh Token으로 세션을 연장하거나 폐기하는 요청을 처리합니다
type SessionHandler struct {
	db            database.DBTX
	refreshConfig auth.RefreshTokenConfig
	limiter       *ratelimit.RateLimiter
	// now는 현재 시각을 반환합니다 (테스트에서 시간을 고정하기 위해 교체할 수 있음)
	now func() time.Time
}

// NewSessionHandler는 SessionHandler의 새 인스턴스를 생성합니다
func NewSessionHandler(db database.DBTX) *SessionHandler {
	return &SessionHandler{
		db:            db,
		refreshConfig: auth.LoadRefreshTokenConfig(),
		limiter:       ratelimit.NewRateLimiter(rate.Every(refreshRateInterval), refreshRateBurst),
		now:           time.Now,
	}
}

// Refresh는 Refresh Token을 한 번 사용하고 새 Access Token과 Refresh Token을 발급합니다
//
// @Summary      어드민 토큰 갱신
// @Description  Refresh Token을 회전해 새 토큰 쌍을 발급합니다. 사용된 토큰을 유예 시간(30초) 밖에서 다시 제출하면 세션 전체가 폐기됩니다.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshTokenRequest true "Refresh Token"
// @Success      200 {object} TokenPairResponse
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "INVALID_REFRESH_TOKEN, REFRESH_TOKEN_EXPIRED, REFRESH_TOKEN_REUSED, USER_NOT_FOUND"
// @Failure      429 {object} ErrorResponse "RATE_LIMIT_EXCEEDED"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Router       /auth/refresh [post]
func (h *SessionHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, ok := decodeRefreshTokenRequest(w, r)
	if !ok {
		return
	}

	pair, err := rotateSession(r.Context(), h.db, refreshToken, h.refreshConfig, h.limiter, h.now())
	if err != nil {
		respondRefreshError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, pair)
}

// Logout은 Refresh Token이 속한 세션(계열)을 폐기합니다
// 토큰이 없거나 이미 폐기되었어도 204를 반환합니다 (토큰 존재 여부를 알려주지 않기 위함)
//
// @Summary      어드민 로그아웃
// @Description  Refresh Token이 속한 세션을 폐기합니다. 이미 발급된 Access Token은 만료될 때까지 유효합니다.
// @Tags         auth
// @Accept       json
// @Param        request body RefreshTokenRequest true "Refresh Token"
// @Success      204 "No Content"
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Router       /auth/logout [post]
func (h *SessionHandler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken, ok := decodeRefreshTokenRequest(w, r)
	if !ok {
		return
	}

	if err := revokeSession(r.Context(), h.db, refreshToken, h.now()); err != nil {
		log.Printf("[ERROR] failed to revoke session: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to revoke session", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// decodeRefreshTokenRequest는 요청 본문에서 refresh_token을 꺼냅니다
// 본문이 잘못되었으면 400 응답을 쓰고 false를 반환합니다
func decodeRefreshTokenRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req RefreshTokenRequest
	body := http.MaxBytesReader(w, r.Body, maxSessionRequestBytes)
	if err := json.NewDecoder(body).Decode(&req); err != nil || req.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "refresh_token is required", nil)
		return "", false
	}
	return req.RefreshToken, true
}

// respondRefreshError는 세션 규칙 에러를 HTTP 상태와 에러 코드로 바꿔 응답합니다
func respondRefreshError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errSessionInvalid):
		respondError(w, http.StatusUnauthorized, ErrInvalidRefreshToken, "Refresh token is invalid or revoked", nil)
	case errors.Is(err, errSessionExpired):
		respondError(w, http.StatusUnauthorized, ErrRefreshTokenExpired, "Refresh token has expired", nil)
	case errors.Is(err, errSessionReused):
		respondError(w, http.StatusUnauthorized, ErrRefreshTokenReused, "Refresh token was reused; the session has been revoked", nil)
	case errors.Is(err, errSessionUserMissing):
		respondError(w, http.StatusUnauthorized, ErrUserNotFound, "User not found", nil)
	case errors.Is(err, errSessionRateLimited):
		w.Header().Set("Retry-After", strconv.Itoa(int(refreshRateInterval.Seconds())))
		respondError(w, http.StatusTooManyRequests, ErrRateLimitExceeded, "Too many refresh requests", nil)
	default:
		log.Printf("[ERROR] failed to refresh session: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to refresh session", nil)
	}
}
```

- [ ] **Step 4: 통과 확인**

실행: `go test ./internal/handlers/ -run TestSessionHandler -count=1 -v`
기대: 모든 서브테스트 PASS

실행: `go test ./internal/handlers/ -count=1`
기대: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/session_handler.go internal/handlers/session_handler_test.go
git commit -m "feat: 토큰 갱신·로그아웃 API 추가"
```

---

### Task 9: 라우트 등록, 비밀키 시작 검증, swagger

**Files:**
- Modify: `cmd/api/main.go` (DB 연결 직후, 핸들러 초기화, `/auth` 라우트 그룹)
- Test: `cmd/api/main_test.go` (끝에 추가)
- Local: `.env` (커밋 대상 아님)

- [ ] **Step 1: 실패하는 테스트 작성**

`cmd/api/main_test.go` import에 `"strings"`와 `"github.com/joho/godotenv"`를 추가하고, 파일 끝에 추가한다.

```go
// TestRun_InvalidRefreshTokenSecret_ReturnsError는 REFRESH_TOKEN_SECRET이 잘못되면 서버가 시작되지 않는지 테스트합니다
func TestRun_InvalidRefreshTokenSecret_ReturnsError(t *testing.T) {
	// Given: 연결 가능한 DB와 비어 있는 비밀키
	_ = godotenv.Load("../../.env")
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}
	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("REFRESH_TOKEN_SECRET", "")

	// When: run() 호출
	err := run()

	// Then: 비밀키 에러로 시작 실패
	if err == nil {
		t.Fatal("expected error when REFRESH_TOKEN_SECRET is invalid, got nil")
	}
	if !strings.Contains(err.Error(), "REFRESH_TOKEN_SECRET") {
		t.Errorf("expected REFRESH_TOKEN_SECRET error, got: %v", err)
	}
}
```

- [ ] **Step 2: 실패 확인**

실행: `go test ./cmd/api/ -run TestRun_InvalidRefreshTokenSecret -count=1 -timeout 30s`
기대: FAIL. 검증이 없어 서버가 8080에서 뜨고 테스트가 멈춘 뒤 timeout(또는 포트 충돌 에러로 메시지 불일치). timeout이면 정상적인 실패로 본다.

- [ ] **Step 3: 구현**

`cmd/api/main.go` import에 `"github.com/june20516/orbithall/internal/auth"`를 추가한다.

`log.Println("Database connected successfully")` 바로 아래에 추가한다.

```go

	// ============================================
	// 인증 설정 검증
	// ============================================
	// Refresh Token 회전에 쓰는 비밀키가 없으면 로그인 후 토큰 갱신이 모두 실패하므로 시작 단계에서 막습니다
	if err := auth.ValidateRefreshTokenSecret(); err != nil {
		return err
	}
```

핸들러 초기화 블록에 추가한다.

```go
	sessionHandler := handlers.NewSessionHandler(db)
```

`/auth` 라우트 그룹을 다음으로 바꾼다.

```go
	// Auth 라우트 그룹 (/auth 접두사, 인증 불필요)
	r.Route("/auth", func(r chi.Router) {
		// Google OAuth 검증 및 토큰 발급
		r.Post("/google/verify", authHandler.GoogleVerify)
		// Refresh Token으로 토큰 쌍 재발급 (계열당 분당 10회 제한)
		r.Post("/refresh", sessionHandler.Refresh)
		// Refresh Token이 속한 세션 폐기
		r.Post("/logout", sessionHandler.Logout)
	})
```

- [ ] **Step 4: 로컬 `.env`에 비밀키 추가**

`.env`에 `REFRESH_TOKEN_SECRET`이 없으면 추가한다(`.env`는 커밋하지 않음).

```bash
grep -q '^REFRESH_TOKEN_SECRET=' .env || echo "REFRESH_TOKEN_SECRET=$(openssl rand -base64 48 | tr -d '\n')" >> .env
```

- [ ] **Step 5: 통과 확인**

실행: `go test ./cmd/api/ -count=1`
기대: `ok`

실행: `go build ./... && go vet ./...`
기대: 에러 없음

- [ ] **Step 6: swagger 재생성 확인 (커밋 안 함)**

실행: `~/go/bin/swag init -g cmd/api/main.go --output ./docs`
기대: 에러 없이 생성

실행: `grep -n '"/auth/refresh"\|"/auth/logout"\|"handlers.TokenPairResponse"' docs/swagger.json`
기대: 세 항목 모두 출력

실행: `git status --short docs/`
기대: `docs/docs.go`, `docs/swagger.*`가 목록에 없음 (`.gitignore` 대상)

- [ ] **Step 7: 로컬 서버 수동 확인**

실행: `docker compose up -d --build api && sleep 5 && curl -s -X POST localhost:8080/auth/refresh -H 'Content-Type: application/json' -d '{"refresh_token":"ohrt_unknown"}'`
기대: `{"error":{"code":"INVALID_REFRESH_TOKEN","message":"Refresh token is invalid or revoked"}}`

실행: `curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8080/auth/logout -H 'Content-Type: application/json' -d '{"refresh_token":"ohrt_unknown"}'`
기대: `204`

실행: `curl -s localhost:8080/admin/profile`
기대: `{"error":{"code":"MISSING_TOKEN","message":"Authorization header is required"}}`

- [ ] **Step 8: Commit**

```bash
git add cmd/api/main.go cmd/api/main_test.go
git commit -m "feat: 토큰 갱신·로그아웃 라우트 등록과 비밀키 시작 검증"
```

---

### Task 10: README와 후속 작업 문서

**Files:**
- Modify: `README.md` (Admin API 인증 절, 환경변수 표, Rate Limiting 제한 정책)
- Create: `docs/tasks/pending/p1-022-admin-token-legacy-cleanup.md`

- [ ] **Step 1: README Admin API 인증 절 수정**

`README.md`의 `#### 인증` 절(``POST /auth/google/verify`` 코드 블록과 요청 예시)을 다음으로 바꾼다.

````markdown
#### 인증

로그인하면 Access Token(7일)과 Refresh Token(유휴 14일, 최대 30일)을 발급합니다. 상세 규칙은 `docs/specs/admin-auth-token-refresh.md`를 참고하세요.

```
POST /auth/google/verify   # Google ID Token 검증 후 토큰 쌍 발급
POST /auth/refresh         # Refresh Token 회전, 새 토큰 쌍 발급
POST /auth/logout          # Refresh Token이 속한 세션 폐기 (204)
```

로그인 요청 예시:
```json
{
  "id_token": "Google OAuth ID Token",
  "email": "user@example.com",
  "name": "사용자 이름"
}
```

로그인·갱신 응답의 토큰 필드:
```json
{
  "token_type": "Bearer",
  "access_token": "eyJ...",
  "access_token_expires_at": "2026-10-08T12:00:00Z",
  "refresh_token": "ohrt_...",
  "refresh_token_expires_at": "2026-10-15T12:00:00Z"
}
```

갱신·로그아웃 요청 본문은 `{"refresh_token": "ohrt_..."}`입니다. 인증 에러는 `{"error":{"code","message"}}` 형식입니다.
````

- [ ] **Step 2: README 환경변수 표에 행 추가**

`## 환경변수` 표의 `ENV` 행 아래에 추가한다.

```markdown
| `JWT_SECRET`                 | Access Token 서명 키 (32자 이상)                        | (필수)                       |
| `JWT_EXPIRATION_HOURS`       | Access Token 수명(시간)                                  | `168`                        |
| `REFRESH_TOKEN_SECRET`       | 후속 Refresh Token 파생 키 (32자 이상, JWT_SECRET과 다름) | (필수)                       |
| `REFRESH_TOKEN_IDLE_TTL`     | Refresh Token 유휴 만료                                  | `336h`                       |
| `REFRESH_TOKEN_ABSOLUTE_TTL` | 세션 절대 만료                                           | `720h`                       |
| `REFRESH_TOKEN_REUSE_GRACE`  | 사용된 Refresh Token 재제출 유예 시간                    | `30s`                        |
```

- [ ] **Step 3: README Rate Limiting 제한 정책에 줄 추가**

`### 제한 정책`의 `- **댓글 작성**: 10회/분 (burst: 5)` 아래에 추가한다.

```markdown
- **어드민 토큰 갱신**: 로그인 세션(계열)당 10회/분 (burst: 10), IP 기준 아님
```

- [ ] **Step 4: 전환 3단계 후속 작업 문서 작성**

`docs/tasks/pending/p1-022-admin-token-legacy-cleanup.md`:

```markdown
# 어드민 토큰 전환 3단계 (호환 코드 제거)

## 작성일
2026-09-28

## 우선순위
- [x] 높음

## 작업 개요
orbithall-admin이 `access_token`을 읽도록 바뀐 뒤, 로그인 응답의 호환용 `token` 필드를 제거한다. 명세 `docs/specs/admin-auth-token-refresh.md` 7장 3단계다.

## 작업 범위
### 포함
- 로그인 응답의 `token` 필드 제거 (`GoogleVerifyResponse.Token`)
- README·명세 갱신

### 제외
- `/admin/*` 핸들러 본문의 평문 에러 통일 (별도 작업)

## 착수 조건
- orbithall-admin이 `token` 대신 `access_token`을 읽도록 배포 완료

## 의존성
- 선행: 021, orbithall-admin 토큰 갱신 구현
```

- [ ] **Step 5: Commit**

```bash
git add README.md docs/tasks/pending/p1-022-admin-token-legacy-cleanup.md
git commit -m "docs: 토큰 갱신 API 문서와 전환 3단계 작업 추가"
```

---

### Task 11: 전체 검증과 작업 완료 기록

**Files:**
- Move: `docs/tasks/active/021-admin-token-refresh.md` → `docs/tasks/completed/021-admin-token-refresh.md`

- [ ] **Step 1: 전체 테스트와 정적 검사**

실행: `go test ./... -count=1`
기대: 모든 패키지 `ok` (DB 통합 테스트가 SKIP 없이 실행되었는지 `-v`로 한 번 확인)

실행: `go vet ./... && ~/go/bin/staticcheck ./...`
기대: 출력 없음

실행: `docker build --target production .`
기대: 빌드 성공 (CI의 PR 검증과 같은 명령)

- [ ] **Step 2: 명세 대조**

`docs/specs/admin-auth-token-refresh.md`의 2장(API)·3장(회전 규칙)·5장(보안)·8장(테스트 시나리오)을 훑어 구현과 다른 점이 없는지 확인한다. 다른 점이 있으면 구현을 고치거나, 의도한 변경이면 명세 변경 이력에 기록한다.

- [ ] **Step 3: 작업 문서 완료 처리**

`docs/tasks/active/021-admin-token-refresh.md`의 제목에서 `[WIP] `를 지우고, `## 작업 이력`에 추가한다.

```markdown
### [완료일] 구현 완료
- refresh_tokens 테이블(006), /auth/refresh, /auth/logout 추가
- 로그인 응답에 토큰 쌍 추가, 인증 에러 본문 객체 형식 통일
- 후속: p1-022 (전환 3단계), orbithall-admin 갱신 구현
```

`[완료일]`은 실제 완료 날짜(YYYY-MM-DD)로 쓴다.

```bash
git mv docs/tasks/active/021-admin-token-refresh.md docs/tasks/completed/021-admin-token-refresh.md
```

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/completed/021-admin-token-refresh.md
git commit -m "docs: 021 어드민 토큰 갱신 작업 완료 기록"
```

- [ ] **Step 5: 배포 전 확인 사항 전달**

사용자에게 다음을 알린다.
- Render에 `REFRESH_TOKEN_SECRET`을 설정해야 한다. 없으면 서버가 시작되지 않는다.
- 마이그레이션 006은 컨테이너 시작 시 `entrypoint.sh`가 자동으로 적용한다.
- 배포 즉시 기존 Access Token이 무효(401 `INVALID_TOKEN`)가 되어 로그인 사용자 전원이 한 번 재로그인해야 한다. 그 외에는 기존 클라이언트와 호환된다(`token` 필드 유지, 인증 에러 상태 코드 동일).
