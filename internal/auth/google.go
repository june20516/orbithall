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

	// Claims에서 사용자 정보 추출
	googleID, ok := payload.Claims["sub"].(string)
	if !ok || googleID == "" {
		return nil, ErrInvalidIDToken
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	return &GoogleIDTokenPayload{
		GoogleID: googleID,
		Email:    email,
		Name:     name,
		Picture:  picture,
	}, nil
}
