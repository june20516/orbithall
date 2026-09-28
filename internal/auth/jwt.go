package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
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

const (
	// AccessTokenType은 어드민 API 호출용 Access Token의 typ 클레임 값입니다
	AccessTokenType = "access"

	// TokenIssuer는 이 서버가 발급한 토큰임을 나타내는 iss 클레임 값입니다
	TokenIssuer = "orbithall"

	// AdminAudience는 토큰을 받는 대상(어드민 API)을 나타내는 aud 클레임 값입니다
	AdminAudience = "orbithall-admin"

	// defaultAccessTokenExpirationHours는 JWT_EXPIRATION_HOURS가 없거나 잘못되었을 때 쓰는 Access Token 수명(7일)입니다
	defaultAccessTokenExpirationHours = 168
)

// CustomClaims는 JWT 토큰에 포함될 사용자 정의 클레임입니다
type CustomClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`

	// TokenType은 토큰 용도입니다 (Access Token은 AccessTokenType)
	TokenType string `json:"typ,omitempty"`

	jwt.RegisteredClaims
}

// GenerateJWT는 Access Token을 발급하고 토큰 문자열만 반환합니다
// 만료 시각이 필요하면 GenerateAccessToken을 사용합니다
func GenerateJWT(userID int64, email string) (string, error) {
	token, _, err := GenerateAccessToken(userID, email)
	return token, err
}

// GenerateAccessToken은 사용자 ID와 이메일을 담은 Access Token과 그 만료 시각을 반환합니다
// JWT_SECRET과 JWT_EXPIRATION_HOURS 환경변수를 사용합니다
func GenerateAccessToken(userID int64, email string) (string, time.Time, error) {
	if err := ValidateJWTSecret(); err != nil {
		return "", time.Time{}, err
	}
	jwtSecret := os.Getenv("JWT_SECRET")

	expirationHours := accessTokenExpirationHours()

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

// ValidateJWTSecret은 JWT_SECRET 환경변수가 비어 있지 않고 32자 이상인지 확인합니다
// 서버 시작 시 호출해 설정 누락을 조기에 발견합니다
func ValidateJWTSecret() error {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if len(jwtSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}
	return nil
}

// accessTokenExpirationHours는 JWT_EXPIRATION_HOURS 환경변수에서 Access Token 수명(시간)을 읽습니다
// 없으면 기본값을 쓰고, 값이 있는데 정수가 아니거나 0 이하이면 경고를 남기고 기본값을 씁니다
func accessTokenExpirationHours() int {
	value := os.Getenv("JWT_EXPIRATION_HOURS")
	if value == "" {
		return defaultAccessTokenExpirationHours
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		log.Printf("[WARN] JWT_EXPIRATION_HOURS=%q is invalid, using default %d", value, defaultAccessTokenExpirationHours)
		return defaultAccessTokenExpirationHours
	}

	return parsed
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
	// iss·aud·exp는 필수이며, 값이 없거나 이 서버·어드민 API용이 아니면 거부합니다
	// 서명 알고리즘은 HS256만 허용합니다
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// HMAC 서명 방식인지 확인
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	}, jwt.WithIssuer(TokenIssuer), jwt.WithAudience(AdminAudience),
		jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		// 서명이 유효하고 만료된 토큰은 다른 클레임 오류(iss·aud·typ 등)와 관계없이
		// 만료로 응답합니다 (클라이언트가 Refresh Token으로 갱신을 시도하도록)
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

	// Access Token 용도의 토큰만 받습니다
	if claims.TokenType != AccessTokenType {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
