package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func init() {
	// 테스트용 환경변수 설정
	os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long-for-security")
	os.Setenv("JWT_EXPIRATION_HOURS", "168")
}

// TestGenerateJWT는 JWT 생성을 테스트합니다
func TestGenerateJWT(t *testing.T) {
	t.Run("JWT 생성 성공", func(t *testing.T) {
		// Given: 사용자 정보
		userID := int64(123)
		email := "test@example.com"

		// When: JWT 생성
		token, err := GenerateJWT(userID, email)

		// Then: 토큰 생성 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if token == "" {
			t.Fatal("expected non-empty token")
		}

		// JWT 형식 확인 (헤더.페이로드.서명)
		// JWT는 세 부분으로 구성되며 점(.)으로 구분됨
		// 최소한 2개 이상의 점이 있어야 함
		dotCount := 0
		for _, c := range token {
			if c == '.' {
				dotCount++
			}
		}
		if dotCount < 2 {
			t.Errorf("expected JWT format with at least 2 dots, got %d dots", dotCount)
		}
	})

	t.Run("JWT_SECRET이 없으면 에러", func(t *testing.T) {
		// Given: JWT_SECRET 환경변수 제거
		original := os.Getenv("JWT_SECRET")
		os.Unsetenv("JWT_SECRET")
		defer os.Setenv("JWT_SECRET", original)

		// When: JWT 생성 시도
		_, err := GenerateJWT(1, "test@example.com")

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error when JWT_SECRET is not set, got nil")
		}
	})

	t.Run("JWT_SECRET이 32자 미만이면 에러", func(t *testing.T) {
		// Given: 짧은 JWT_SECRET
		original := os.Getenv("JWT_SECRET")
		os.Setenv("JWT_SECRET", "short-key")
		defer os.Setenv("JWT_SECRET", original)

		// When: JWT 생성 시도
		_, err := GenerateJWT(1, "test@example.com")

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for short JWT_SECRET, got nil")
		}
	})
}

// TestValidateJWT는 JWT 검증을 테스트합니다
func TestValidateJWT(t *testing.T) {
	t.Run("유효한 JWT 검증 성공", func(t *testing.T) {
		// Given: 생성된 JWT
		userID := int64(456)
		email := "validate@example.com"
		token, err := GenerateJWT(userID, email)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		// When: JWT 검증
		claims, err := ValidateJWT(token)

		// Then: 검증 성공 및 claims 확인
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if claims == nil {
			t.Fatal("expected claims, got nil")
		}
		if claims.UserID != userID {
			t.Errorf("expected user_id=%d, got %d", userID, claims.UserID)
		}
		if claims.Email != email {
			t.Errorf("expected email=%s, got %s", email, claims.Email)
		}
	})

	t.Run("잘못된 토큰 검증 실패", func(t *testing.T) {
		// Given: 잘못된 토큰
		invalidToken := "invalid.token.here"

		// When: JWT 검증
		_, err := ValidateJWT(invalidToken)

		// Then: ErrInvalidToken 반환
		if err == nil {
			t.Fatal("expected error for invalid token, got nil")
		}
		if err != ErrInvalidToken {
			t.Errorf("expected ErrInvalidToken, got: %v", err)
		}
	})

	t.Run("빈 토큰 검증 실패", func(t *testing.T) {
		// Given: 빈 토큰
		emptyToken := ""

		// When: JWT 검증
		_, err := ValidateJWT(emptyToken)

		// Then: ErrInvalidToken 반환
		if err == nil {
			t.Fatal("expected error for empty token, got nil")
		}
		if err != ErrInvalidToken {
			t.Errorf("expected ErrInvalidToken, got: %v", err)
		}
	})

	t.Run("잘못된 서명 검증 실패", func(t *testing.T) {
		// Given: 유효한 JWT 생성
		token, err := GenerateJWT(789, "wrong@example.com")
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		// JWT_SECRET 변경
		original := os.Getenv("JWT_SECRET")
		os.Setenv("JWT_SECRET", "different-secret-key-at-least-32-characters-long")
		defer os.Setenv("JWT_SECRET", original)

		// When: 다른 시크릿으로 검증
		_, err = ValidateJWT(token)

		// Then: ErrInvalidToken 반환
		if err == nil {
			t.Fatal("expected error for wrong signature, got nil")
		}
		if err != ErrInvalidToken {
			t.Errorf("expected ErrInvalidToken, got: %v", err)
		}
	})
}

// TestValidateJWT_Expiration은 만료된 토큰 검증을 테스트합니다
// 서명이 유효하고 만료된 토큰은 다른 클레임 오류(iss 누락, typ 불일치 등)와
// 관계없이 항상 ErrExpiredToken을 반환해야 합니다
func TestValidateJWT_Expiration(t *testing.T) {
	t.Run("만료된 토큰은 ErrExpiredToken", func(t *testing.T) {
		// Given: 만료 시각이 과거인 클레임
		claims := validTestClaims()
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
		token := signTestClaims(t, claims)

		// When: 검증
		_, err := ValidateJWT(token)

		// Then: ErrExpiredToken
		if err != ErrExpiredToken {
			t.Errorf("expected ErrExpiredToken, got: %v", err)
		}
	})

	tests := []struct {
		name   string
		mutate func(claims *CustomClaims)
	}{
		{name: "만료 + iss 누락", mutate: func(c *CustomClaims) { c.Issuer = "" }},
		{name: "만료 + typ 불일치(refresh)", mutate: func(c *CustomClaims) { c.TokenType = "refresh" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 만료 시각이 과거이면서 다른 필수 클레임도 잘못된 토큰
			claims := validTestClaims()
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
			tt.mutate(claims)
			token := signTestClaims(t, claims)

			// When: 검증
			_, err := ValidateJWT(token)

			// Then: 다른 클레임 오류와 관계없이 ErrExpiredToken
			if err != ErrExpiredToken {
				t.Errorf("expected ErrExpiredToken, got: %v", err)
			}
		})
	}
}

// TestCustomClaims는 CustomClaims 구조체를 테스트합니다
func TestCustomClaims(t *testing.T) {
	t.Run("CustomClaims 생성 및 검증", func(t *testing.T) {
		// Given: 사용자 정보
		userID := int64(999)
		email := "claims@example.com"

		// When: JWT 생성 및 검증
		token, err := GenerateJWT(userID, email)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		claims, err := ValidateJWT(token)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		// Then: 만료 시간 확인
		expiresAt := claims.ExpiresAt
		if expiresAt == nil || expiresAt.IsZero() {
			t.Error("expected non-zero expiration time")
		}

		// 만료 시간이 미래인지 확인
		if expiresAt != nil && time.Until(expiresAt.Time) <= 0 {
			t.Error("expected expiration time to be in the future")
		}
	})
}

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
		first, _, err := GenerateAccessToken(42, "access@example.com")
		if err != nil {
			t.Fatalf("failed to generate first token: %v", err)
		}
		second, _, err := GenerateAccessToken(42, "access@example.com")
		if err != nil {
			t.Fatalf("failed to generate second token: %v", err)
		}
		firstClaims, err := ValidateJWT(first)
		if err != nil {
			t.Fatalf("failed to validate first token: %v", err)
		}
		secondClaims, err := ValidateJWT(second)
		if err != nil {
			t.Fatalf("failed to validate second token: %v", err)
		}

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
		{name: "exp가 없으면 거부", mutate: func(c *CustomClaims) { c.ExpiresAt = nil }},
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

// TestValidateJWT_RejectsNonHS256Algorithm은 HS256 외 알고리즘으로 서명된
// 토큰을 거부하는지 테스트합니다
func TestValidateJWT_RejectsNonHS256Algorithm(t *testing.T) {
	t.Run("HS512으로 서명된 토큰은 거부", func(t *testing.T) {
		// Given: 같은 비밀키로 HS512 서명한 유효 클레임 토큰
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, validTestClaims()).SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			t.Fatalf("failed to sign test claims: %v", err)
		}

		// When: 검증
		_, err = ValidateJWT(token)

		// Then: ErrInvalidToken
		if err != ErrInvalidToken {
			t.Errorf("expected ErrInvalidToken, got: %v", err)
		}
	})
}
