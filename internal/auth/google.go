package auth

import (
	"context"
	"errors"
	"fmt"
	"os"

	"google.golang.org/api/idtoken"
)

var (
	// ErrInvalidIDToken은 Google ID Token이 유효하지 않을 때 반환됩니다
	ErrInvalidIDToken = errors.New("invalid google id token")
)

// GoogleIDTokenPayload는 Google ID Token에서 추출한 사용자 정보입니다
type GoogleIDTokenPayload struct {
	GoogleID string
	Email    string
	Name     string
	Picture  string
}

// VerifyGoogleIDToken은 Google ID Token을 검증하고 사용자 정보를 추출합니다
// GOOGLE_CLIENT_ID 환경변수가 필수입니다
func VerifyGoogleIDToken(ctx context.Context, idToken string) (*GoogleIDTokenPayload, error) {
	// GOOGLE_CLIENT_ID 검증
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID environment variable is required")
	}

	// Google ID Token 검증
	payload, err := idtoken.Validate(ctx, idToken, clientID)
	if err != nil {
		return nil, ErrInvalidIDToken
	}

	return payloadFromClaims(payload.Claims)
}

// payloadFromClaims는 서명이 검증된 ID Token의 클레임에서 사용자 정보를 추출합니다
// sub(Google 사용자 ID)와 email이 비어 있거나, Google이 이메일 소유를 확인하지 않았으면(email_verified가 true가 아님)
// ErrInvalidIDToken을 반환합니다. name과 picture는 없을 수 있으므로 빈 값을 허용합니다
func payloadFromClaims(claims map[string]interface{}) (*GoogleIDTokenPayload, error) {
	googleID, _ := claims["sub"].(string)
	if googleID == "" {
		return nil, ErrInvalidIDToken
	}

	email, _ := claims["email"].(string)
	if email == "" {
		return nil, ErrInvalidIDToken
	}
	if !isEmailVerified(claims["email_verified"]) {
		return nil, ErrInvalidIDToken
	}

	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)

	return &GoogleIDTokenPayload{
		GoogleID: googleID,
		Email:    email,
		Name:     name,
		Picture:  picture,
	}, nil
}

// isEmailVerified는 email_verified 클레임이 참인지 확인합니다
// Google은 이 값을 불리언(true) 또는 문자열("true")로 보낼 수 있으므로 두 형식을 모두 허용합니다
func isEmailVerified(value interface{}) bool {
	switch verified := value.(type) {
	case bool:
		return verified
	case string:
		return verified == "true"
	default:
		return false
	}
}
