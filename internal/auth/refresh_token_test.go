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
