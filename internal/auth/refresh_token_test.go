package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"
)

// testRefreshTokenSecret은 테스트용 Refresh Token 비밀키입니다 (jwt_test.go의 JWT_SECRET과 다른 값)
const testRefreshTokenSecret = "test-refresh-secret-at-least-32-characters-long"

// testJWTSecretForRefreshValidation은 TestValidateRefreshTokenSecret 전용 JWT_SECRET입니다
// jwt_test.go의 init()에 의존하지 않도록 32자 이상의 고정값을 직접 둡니다
const testJWTSecretForRefreshValidation = "test-jwt-secret-for-refresh-validation-32+"

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

	t.Run("HMAC-SHA256(비밀키, 입력)을 base64url로 인코딩한 값이다", func(t *testing.T) {
		// Given: 입력과 비밀키로 독립적으로 계산한 기대값
		const input = "ohrt_input"
		mac := hmac.New(sha256.New, []byte(testRefreshTokenSecret))
		mac.Write([]byte(input))
		want := RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

		// When: 파생
		got, err := DeriveNextRefreshToken(input)

		// Then: 독립적으로 계산한 값과 일치하고 길이도 일치함
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
		if len(got) != len(RefreshTokenPrefix)+43 {
			t.Errorf("expected length %d, got %d", len(RefreshTokenPrefix)+43, len(got))
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
		name            string
		jwtSecret       string
		secret          string
		wantErrContains string
	}{
		{name: "비어 있으면 에러", jwtSecret: testJWTSecretForRefreshValidation, secret: "", wantErrContains: "at least 32 characters"},
		{name: "32자 미만이면 에러", jwtSecret: testJWTSecretForRefreshValidation, secret: "too-short-secret", wantErrContains: "at least 32 characters"},
		{name: "JWT_SECRET과 같으면 에러", jwtSecret: testJWTSecretForRefreshValidation, secret: testJWTSecretForRefreshValidation, wantErrContains: "differ from JWT_SECRET"},
		{name: "32자 이상이고 JWT_SECRET과 다르면 통과", jwtSecret: testJWTSecretForRefreshValidation, secret: testRefreshTokenSecret, wantErrContains: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: JWT_SECRET과 REFRESH_TOKEN_SECRET 설정
			t.Setenv("JWT_SECRET", tt.jwtSecret)
			t.Setenv("REFRESH_TOKEN_SECRET", tt.secret)

			// When: 검증
			err := ValidateRefreshTokenSecret()

			// Then: 기대한 사유의 에러이거나 에러 없음
			if tt.wantErrContains == "" {
				if err != nil {
					t.Errorf("ValidateRefreshTokenSecret() expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateRefreshTokenSecret() expected error containing %q, got nil", tt.wantErrContains)
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("ValidateRefreshTokenSecret() error = %q, want it to contain %q", err.Error(), tt.wantErrContains)
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

// captureLog는 테스트 동안 표준 logger 출력을 버퍼로 받고, 끝나면 원래 출력으로 되돌립니다
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	original := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(original) })
	return &buf
}

// TestDurationFromEnv_Warning은 잘못된 값으로 기본값을 쓸 때만 경고를 남기는지 테스트합니다
func TestDurationFromEnv_Warning(t *testing.T) {
	const key = "REFRESH_TOKEN_IDLE_TTL"

	t.Run("형식이 잘못되었거나 0 이하이면 경고를 남긴다", func(t *testing.T) {
		for _, value := range []string{"two weeks", "-1h", "0s"} {
			// Given: 잘못된 값과 로그 버퍼
			t.Setenv(key, value)
			buf := captureLog(t)

			// When: 읽기
			got := durationFromEnv(key, time.Hour)

			// Then: 기본값과 경고
			if got != time.Hour {
				t.Errorf("value %q: got %v, want fallback", value, got)
			}
			want := fmt.Sprintf("[WARN] %s=%q is invalid, using default 1h0m0s", key, value)
			if !strings.Contains(buf.String(), want) {
				t.Errorf("value %q: log = %q, want to contain %q", value, buf.String(), want)
			}
		}
	})

	t.Run("정상 값이나 빈 값이면 경고를 남기지 않는다", func(t *testing.T) {
		for _, value := range []string{"48h", ""} {
			// Given: 정상 값 또는 빈 값과 로그 버퍼
			t.Setenv(key, value)
			buf := captureLog(t)

			// When: 읽기
			durationFromEnv(key, time.Hour)

			// Then: 로그 없음
			if buf.Len() != 0 {
				t.Errorf("value %q: unexpected log %q", value, buf.String())
			}
		}
	})
}
