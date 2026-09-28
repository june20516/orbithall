package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"
)

// RefreshTokenPrefix는 Refresh Token 앞에 붙는 식별 접두사입니다
// 토큰이 로그나 저장소에 유출되었을 때 스캐너가 쉽게 찾을 수 있게 합니다
const RefreshTokenPrefix = "ohrt_"

// refreshTokenRandomBytes는 로그인 시 첫 Refresh Token을 만드는 난수 길이입니다
const refreshTokenRandomBytes = 32

// Refresh Token 수명 기본값
const (
	defaultRefreshTokenIdleTTL     = 14 * 24 * time.Hour
	defaultRefreshTokenAbsoluteTTL = 30 * 24 * time.Hour
	defaultRefreshTokenReuseGrace  = 30 * time.Second
)

// RefreshTokenConfig는 Refresh Token 수명 설정입니다
type RefreshTokenConfig struct {
	// IdleTTL은 토큰 발급 후 이 시간 동안 쓰지 않으면 만료되는 기간입니다
	IdleTTL time.Duration

	// AbsoluteTTL은 최초 로그인 후 갱신과 관계없이 세션이 끝나는 기간입니다
	AbsoluteTTL time.Duration

	// ReuseGrace는 사용된 토큰이 다시 제출되어도 재사용으로 보지 않는 유예 시간입니다
	// 동시 요청이나 쿠키 저장 실패로 같은 토큰이 두 번 오는 경우를 흡수합니다
	ReuseGrace time.Duration
}

// LoadRefreshTokenConfig는 환경변수에서 Refresh Token 수명 설정을 읽습니다
// 값은 time.ParseDuration 형식(예: "336h", "30s")이며, 없거나 잘못되면 기본값을 씁니다
func LoadRefreshTokenConfig() RefreshTokenConfig {
	return RefreshTokenConfig{
		IdleTTL:     durationFromEnv("REFRESH_TOKEN_IDLE_TTL", defaultRefreshTokenIdleTTL),
		AbsoluteTTL: durationFromEnv("REFRESH_TOKEN_ABSOLUTE_TTL", defaultRefreshTokenAbsoluteTTL),
		ReuseGrace:  durationFromEnv("REFRESH_TOKEN_REUSE_GRACE", defaultRefreshTokenReuseGrace),
	}
}

// durationFromEnv는 환경변수를 time.Duration으로 읽고, 없거나 0 이하이거나 형식이 틀리면 fallback을 반환합니다
// 값이 있는데 쓸 수 없으면 설정 실수를 알 수 있도록 경고를 남깁니다
func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		log.Printf("[WARN] %s=%q is invalid, using default %s", key, value, fallback)
		return fallback
	}

	return parsed
}

// GenerateRefreshToken은 로그인 시 발급하는 첫 Refresh Token을 만듭니다
// 암호학적 난수 32바이트를 패딩 없는 base64url로 인코딩하고 접두사를 붙입니다
func GenerateRefreshToken() (string, error) {
	randomBytes := make([]byte, refreshTokenRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

// DeriveNextRefreshToken은 회전 시 이전 토큰에서 후속 토큰을 만듭니다
// HMAC-SHA256(REFRESH_TOKEN_SECRET, 이전 토큰 원문)이므로 같은 이전 토큰에서는 항상 같은 후속 토큰이 나옵니다
// 덕분에 원문을 저장하지 않고도 유예 시간 안의 재요청에 같은 후속 토큰을 다시 돌려줄 수 있습니다
// 비밀키 없이는 이전 토큰으로 후속 토큰을 예측할 수 없습니다
func DeriveNextRefreshToken(current string) (string, error) {
	secret, err := refreshTokenSecret()
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(current))

	return RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// ValidateRefreshTokenSecret은 REFRESH_TOKEN_SECRET 환경변수가 올바른지 확인합니다
// 서버 시작 시 호출해 설정 누락을 조기에 발견합니다
func ValidateRefreshTokenSecret() error {
	_, err := refreshTokenSecret()
	return err
}

// refreshTokenSecret은 후속 토큰 파생에 쓰는 비밀키를 읽고 검증합니다
// JWT_SECRET과 같은 값을 쓰면 한 키의 유출이 두 용도 모두에 영향을 주므로 거부합니다
func refreshTokenSecret() ([]byte, error) {
	secret := os.Getenv("REFRESH_TOKEN_SECRET")
	if len(secret) < 32 {
		return nil, fmt.Errorf("REFRESH_TOKEN_SECRET must be at least 32 characters long")
	}
	if secret == os.Getenv("JWT_SECRET") {
		return nil, fmt.Errorf("REFRESH_TOKEN_SECRET must differ from JWT_SECRET")
	}

	return []byte(secret), nil
}

// HashRefreshToken은 DB에 저장하고 조회할 때 쓰는 토큰의 SHA-256 해시를 반환합니다
// 토큰 자체가 충분한 난수이므로 salt 없는 단일 해시로 충분합니다
func HashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
