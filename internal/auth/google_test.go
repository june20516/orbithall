package auth

import (
	"context"
	"os"
	"testing"
)

// TestVerifyGoogleIDToken_EnvCheck는 환경변수 검증을 테스트합니다
func TestVerifyGoogleIDToken_EnvCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("GOOGLE_CLIENT_ID가 없으면 에러", func(t *testing.T) {
		// Given: GOOGLE_CLIENT_ID 환경변수 제거
		original := os.Getenv("GOOGLE_CLIENT_ID")
		os.Unsetenv("GOOGLE_CLIENT_ID")
		defer os.Setenv("GOOGLE_CLIENT_ID", original)

		// When: VerifyGoogleIDToken 호출
		_, err := VerifyGoogleIDToken(ctx, "some-token")

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error when GOOGLE_CLIENT_ID is not set, got nil")
		}
	})

	t.Run("잘못된 토큰은 ErrInvalidIDToken 반환", func(t *testing.T) {
		// Given: GOOGLE_CLIENT_ID 설정 (테스트용 더미 값)
		os.Setenv("GOOGLE_CLIENT_ID", "test-client-id.apps.googleusercontent.com")

		// When: 잘못된 토큰으로 검증
		_, err := VerifyGoogleIDToken(ctx, "invalid-token")

		// Then: ErrInvalidIDToken 반환
		if err != ErrInvalidIDToken {
			t.Errorf("expected ErrInvalidIDToken, got: %v", err)
		}
	})
}

// 참고: 실제 Google ID Token 검증 테스트는 통합 테스트에서 수행
// 실제 토큰을 생성하려면 Google OAuth Playground를 사용해야 함
