package auth

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken은 JWT 토큰이 유효하지 않을 때 반환됩니다
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken은 JWT 토큰이 만료되었을 때 반환됩니다
	ErrExpiredToken = errors.New("expired token")
)

// CustomClaims는 JWT 토큰에 포함될 사용자 정의 클레임입니다
type CustomClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateJWT는 사용자 ID와 이메일을 포함하는 JWT를 생성합니다
// JWT_SECRET과 JWT_EXPIRATION_HOURS 환경변수를 사용합니다
func GenerateJWT(userID int64, email string) (string, error) {
	// JWT_SECRET 검증
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return "", fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if len(jwtSecret) < 32 {
		return "", fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	// JWT_EXPIRATION_HOURS 읽기 (기본값: 168시간 = 7일)
	expirationHours := 168
	if expirationStr := os.Getenv("JWT_EXPIRATION_HOURS"); expirationStr != "" {
		if parsed, err := strconv.Atoi(expirationStr); err == nil {
			expirationHours = parsed
		}
	}

	// 만료 시간 계산
	expiresAt := time.Now().Add(time.Duration(expirationHours) * time.Hour)

	// Claims 생성
	claims := &CustomClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// JWT 토큰 생성 (HS256 알고리즘)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 서명하여 문자열로 변환
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT는 JWT 토큰을 검증하고 claims를 반환합니다
func ValidateJWT(tokenString string) (*CustomClaims, error) {
	// 빈 토큰 체크
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	// JWT_SECRET 가져오기
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	// 토큰 파싱 및 검증
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// HMAC 서명 방식인지 확인
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		// 만료된 토큰 체크
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	// Claims 추출
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
