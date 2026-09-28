package auth

import (
	"context"
	"errors"
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

// validGoogleClaims는 필수 클레임이 모두 채워진 ID Token 클레임을 만듭니다
func validGoogleClaims() map[string]interface{} {
	return map[string]interface{}{
		"sub":            "google-sub-123",
		"email":          "user@example.com",
		"email_verified": true,
		"name":           "Google User",
		"picture":        "https://example.com/pic.jpg",
	}
}

// TestPayloadFromClaims는 검증된 ID Token 클레임에서 사용자 정보를 추출하는 규칙을 테스트합니다
func TestPayloadFromClaims(t *testing.T) {
	t.Run("필수 클레임이 모두 있으면 사용자 정보를 추출한다", func(t *testing.T) {
		// Given: 유효한 클레임
		claims := validGoogleClaims()

		// When: 추출
		payload, err := payloadFromClaims(claims)

		// Then: 모든 필드가 클레임 값과 같음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		want := GoogleIDTokenPayload{
			GoogleID: "google-sub-123",
			Email:    "user@example.com",
			Name:     "Google User",
			Picture:  "https://example.com/pic.jpg",
		}
		if *payload != want {
			t.Errorf("payload = %+v, want %+v", *payload, want)
		}
	})

	t.Run("email_verified가 문자열 true여도 허용한다", func(t *testing.T) {
		// Given: email_verified가 문자열인 클레임
		claims := validGoogleClaims()
		claims["email_verified"] = "true"

		// When: 추출
		payload, err := payloadFromClaims(claims)

		// Then: 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if payload.Email != "user@example.com" {
			t.Errorf("Email = %q", payload.Email)
		}
	})

	t.Run("name과 picture가 없어도 성공하고 빈 값으로 둔다", func(t *testing.T) {
		// Given: 선택 클레임이 없는 클레임
		claims := validGoogleClaims()
		delete(claims, "name")
		delete(claims, "picture")

		// When: 추출
		payload, err := payloadFromClaims(claims)

		// Then: 성공하고 이름·사진은 빈 값
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if payload.Name != "" || payload.Picture != "" {
			t.Errorf("expected empty name and picture, got %+v", payload)
		}
	})

	invalidCases := []struct {
		name   string
		mutate func(claims map[string]interface{})
	}{
		{name: "sub 누락", mutate: func(c map[string]interface{}) { delete(c, "sub") }},
		{name: "sub가 빈 문자열", mutate: func(c map[string]interface{}) { c["sub"] = "" }},
		{name: "email 누락", mutate: func(c map[string]interface{}) { delete(c, "email") }},
		{name: "email이 빈 문자열", mutate: func(c map[string]interface{}) { c["email"] = "" }},
		{name: "email_verified 누락", mutate: func(c map[string]interface{}) { delete(c, "email_verified") }},
		{name: "email_verified가 false", mutate: func(c map[string]interface{}) { c["email_verified"] = false }},
		{name: "email_verified가 문자열 false", mutate: func(c map[string]interface{}) { c["email_verified"] = "false" }},
		{name: "email_verified가 숫자", mutate: func(c map[string]interface{}) { c["email_verified"] = 1 }},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name+"이면 ErrInvalidIDToken", func(t *testing.T) {
			// Given: 조건을 만족하지 않는 클레임
			claims := validGoogleClaims()
			tc.mutate(claims)

			// When: 추출
			payload, err := payloadFromClaims(claims)

			// Then: ErrInvalidIDToken
			if !errors.Is(err, ErrInvalidIDToken) {
				t.Errorf("expected ErrInvalidIDToken, got payload %+v, err %v", payload, err)
			}
		})
	}
}
