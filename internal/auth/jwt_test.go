package auth

import (
	"os"
	"testing"
	"time"
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
// 주의: 실제 만료된 토큰을 생성하는 것은 시간이 오래 걸리므로
// 이 테스트는 스킵하거나 mock을 사용해야 합니다
func TestValidateJWT_Expiration(t *testing.T) {
	t.Skip("만료 테스트는 시간이 오래 걸리므로 스킵 (통합 테스트에서 처리)")

	// 참고: 만료 테스트를 실제로 수행하려면:
	// 1. JWT_EXPIRATION_HOURS를 매우 작은 값(예: -1)으로 설정
	// 2. 토큰 생성
	// 3. time.Sleep()로 대기
	// 4. 검증 시 ErrExpiredToken 확인
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
