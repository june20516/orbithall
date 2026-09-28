package handlers

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/ratelimit"
)

// staleRefreshTokenRetention은 절대 만료되거나 폐기된 Refresh Token을 삭제하기 전까지 보관하는 기간입니다
// 폐기 직후의 재사용 시도를 조사할 수 있도록 바로 지우지 않습니다
const staleRefreshTokenRetention = 7 * 24 * time.Hour

// 세션 규칙 위반 에러
// SessionHandler가 각 에러를 HTTP 상태와 에러 코드로 바꿉니다
var (
	errSessionInvalid     = errors.New("refresh token is invalid or revoked")
	errSessionExpired     = errors.New("refresh token has expired")
	errSessionReused      = errors.New("refresh token was reused")
	errSessionUserMissing = errors.New("session user not found")
	errSessionRateLimited = errors.New("too many refresh requests")
)

// TokenPairResponse는 로그인과 토큰 갱신 응답에 공통으로 들어가는 토큰 정보입니다
// @Description Access Token과 Refresh Token 발급 결과 (시각은 RFC 3339, UTC)
type TokenPairResponse struct {
	TokenType             string    `json:"token_type" example:"Bearer"`
	AccessToken           string    `json:"access_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshToken          string    `json:"refresh_token" example:"ohrt_..."`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

// issueSession은 로그인한 사용자에게 새 세션(Access Token + 새 계열의 첫 Refresh Token)을 발급합니다
// 같은 사용자의 오래된 Refresh Token도 함께 정리합니다
func issueSession(ctx context.Context, db database.DBTX, user *models.User, cfg auth.RefreshTokenConfig, now time.Time) (*TokenPairResponse, error) {
	if err := database.DeleteStaleRefreshTokens(ctx, db, user.ID, now.Add(-staleRefreshTokenRetention)); err != nil {
		return nil, err
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	familyExpiresAt := now.Add(cfg.AbsoluteTTL)
	stored, err := database.CreateRefreshTokenFamily(
		ctx, db, user.ID,
		auth.HashRefreshToken(refreshToken),
		refreshTokenExpiry(now, cfg.IdleTTL, familyExpiresAt),
		familyExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	return newTokenPairResponse(user, refreshToken, stored.ExpiresAt)
}

// rotateSession은 제출된 Refresh Token을 한 번 사용 처리하고 새 토큰 쌍을 발급합니다
// 규칙은 docs/specs/admin-auth-token-refresh.md 3장을 따릅니다
func rotateSession(ctx context.Context, db database.DBTX, presented string, cfg auth.RefreshTokenConfig, limiter *ratelimit.RateLimiter, now time.Time) (*TokenPairResponse, error) {
	presentedHash := auth.HashRefreshToken(presented)

	current, err := database.GetRefreshTokenByHash(ctx, db, presentedHash)
	if err != nil {
		return nil, err
	}
	if current == nil || current.RevokedAt != nil {
		return nil, errSessionInvalid
	}

	// 계열(로그인 세션) 단위로 갱신 빈도를 제한합니다
	// 유효한(존재하고 폐기되지 않은) 계열만 제한 대상이므로, 임의 토큰으로 limiter 키를 늘릴 수 없습니다
	// 사용 여부와 만료 확인보다 앞에 두어 재사용 탐지와 만료 응답도 제한에 포함합니다
	// 모든 요청이 어드민 서버 IP 하나에서 오므로 IP 기준 제한은 쓰지 않습니다
	if !limiter.GetLimiter(current.FamilyID).Allow() {
		return nil, errSessionRateLimited
	}

	if current.UsedAt != nil {
		return reissueWithinGrace(ctx, db, current, presented, cfg, now)
	}
	if current.IsExpired(now) {
		return nil, errSessionExpired
	}

	// 실패할 수 있는 작업(사용자 조회, Access Token 발급)은 회전보다 먼저 끝냅니다
	// 회전이 커밋된 뒤 실패하면 클라이언트는 이전 토큰을 유지하는데,
	// 유예 시간이 지난 뒤 그 토큰을 다시 내면 재사용으로 판정되어 정상 세션이 폐기되기 때문입니다
	user, err := database.GetUserByID(ctx, db, current.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errSessionUserMissing
	}
	accessToken, accessExpiresAt, err := auth.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	nextToken, err := auth.DeriveNextRefreshToken(presented)
	if err != nil {
		return nil, err
	}

	next, err := database.RotateRefreshToken(
		ctx, db, current.ID,
		auth.HashRefreshToken(nextToken),
		refreshTokenExpiry(now, cfg.IdleTTL, current.FamilyExpiresAt),
		now,
	)
	if err != nil {
		return nil, err
	}

	if next == nil {
		// 같은 토큰으로 들어온 다른 요청이 먼저 회전했거나, 계열이 폐기되었습니다
		// 미리 만든 Access Token은 버리고, 상태를 다시 읽어 사용 처리된 경우에만 유예 시간 규칙으로 판단합니다
		current, err = database.GetRefreshTokenByHash(ctx, db, presentedHash)
		if err != nil {
			return nil, err
		}
		if current == nil || current.RevokedAt != nil || current.UsedAt == nil {
			return nil, errSessionInvalid
		}
		return reissueWithinGrace(ctx, db, current, presented, cfg, now)
	}

	return assembleTokenPairResponse(accessToken, accessExpiresAt, nextToken, next.ExpiresAt), nil
}

// reissueWithinGrace는 이미 사용된 토큰이 다시 제출되었을 때를 처리합니다
//
// 유예 시간 안이고 후속 토큰이 아직 쓰이지 않았다면 그 후속 토큰을 다시 돌려줍니다
// 새 토큰을 만들지 않으므로 계열이 한 줄로 유지되고, 탈취된 경우에도 나중에 후속 토큰을 쓰는 쪽에서 재사용이 탐지됩니다
// 유예 시간이 지났거나 후속 토큰이 이미 사용되었으면 탈취로 보고 계열 전체를 폐기합니다
// 후속 토큰이 만료되었거나 폐기된 경우, 비밀키가 바뀐 경우는 탈취 신호가 아니므로 계열을 폐기하지 않습니다
func reissueWithinGrace(ctx context.Context, db database.DBTX, used *models.RefreshToken, presented string, cfg auth.RefreshTokenConfig, now time.Time) (*TokenPairResponse, error) {
	// 계열이 폐기되었다면 유예 시간과 관계없이 무효입니다
	// 계열 폐기와 동시에 진행된 회전으로 폐기 표시 없이 남은 토큰이 여기로 올 수 있습니다
	familyRevoked, err := database.IsRefreshTokenFamilyRevoked(ctx, db, used.FamilyID)
	if err != nil {
		return nil, err
	}
	if familyRevoked {
		return nil, errSessionInvalid
	}

	if now.Sub(*used.UsedAt) > cfg.ReuseGrace {
		return nil, revokeFamilyForReuse(ctx, db, used.FamilyID, now)
	}

	child, err := database.GetChildRefreshToken(ctx, db, used.ID)
	if err != nil {
		return nil, err
	}
	if child == nil || child.UsedAt != nil {
		return nil, revokeFamilyForReuse(ctx, db, used.FamilyID, now)
	}
	// 계열 폐기 확인 뒤 로그아웃이 커밋된 경우입니다
	if child.RevokedAt != nil {
		return nil, errSessionInvalid
	}
	// 절대 만료 직전에 회전한 후속 토큰이 유예 시간 안에 만료된 경우입니다
	if child.IsExpired(now) {
		return nil, errSessionExpired
	}

	// 후속 토큰은 이전 토큰에서 결정적으로 만들어지므로 원문을 저장하지 않고도 다시 계산할 수 있습니다
	childToken, err := auth.DeriveNextRefreshToken(presented)
	if err != nil {
		return nil, err
	}

	// 비밀키가 바뀐 뒤라면 계산한 값이 저장된 해시와 달라 돌려줄 수 없습니다
	if !bytes.Equal(auth.HashRefreshToken(childToken), child.TokenHash) {
		return nil, errSessionInvalid
	}

	return buildSessionResponse(ctx, db, child.UserID, childToken, child.ExpiresAt)
}

// revokeFamilyForReuse는 재사용 탐지로 계열 전체를 폐기하고 errSessionReused를 반환합니다
func revokeFamilyForReuse(ctx context.Context, db database.DBTX, familyID string, now time.Time) error {
	if err := database.RevokeRefreshTokenFamily(ctx, db, familyID, models.RefreshTokenRevokedByReuse, now); err != nil {
		return err
	}
	return errSessionReused
}

// revokeSession은 제출된 Refresh Token이 속한 계열을 폐기합니다 (로그아웃)
// 토큰이 없거나 이미 폐기된 경우에도 에러 없이 끝납니다
func revokeSession(ctx context.Context, db database.DBTX, presented string, now time.Time) error {
	current, err := database.GetRefreshTokenByHash(ctx, db, auth.HashRefreshToken(presented))
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}

	return database.RevokeRefreshTokenFamily(ctx, db, current.FamilyID, models.RefreshTokenRevokedByLogout, now)
}

// buildSessionResponse는 토큰 소유자를 조회해 새 Access Token과 함께 응답을 만듭니다
func buildSessionResponse(ctx context.Context, db database.DBTX, userID int64, refreshToken string, refreshExpiresAt time.Time) (*TokenPairResponse, error) {
	user, err := database.GetUserByID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errSessionUserMissing
	}

	return newTokenPairResponse(user, refreshToken, refreshExpiresAt)
}

// newTokenPairResponse는 Access Token을 새로 발급해 토큰 쌍 응답을 만듭니다
func newTokenPairResponse(user *models.User, refreshToken string, refreshExpiresAt time.Time) (*TokenPairResponse, error) {
	accessToken, accessExpiresAt, err := auth.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return assembleTokenPairResponse(accessToken, accessExpiresAt, refreshToken, refreshExpiresAt), nil
}

// assembleTokenPairResponse는 이미 발급한 토큰 값들로 토큰 쌍 응답을 만듭니다
// 시각은 UTC로 맞추고, Refresh Token 만료는 초 단위로 자릅니다
// (DB 값의 마이크로초는 버리므로 실제 만료보다 최대 1초 이르게 표시됩니다)
func assembleTokenPairResponse(accessToken string, accessExpiresAt time.Time, refreshToken string, refreshExpiresAt time.Time) *TokenPairResponse {
	return &TokenPairResponse{
		TokenType:             "Bearer",
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt.UTC(),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshExpiresAt.UTC().Truncate(time.Second),
	}
}

// refreshTokenExpiry는 새 Refresh Token의 만료 시각을 계산합니다
// 유휴 만료(now + idleTTL)가 계열의 절대 만료보다 늦으면 절대 만료를 씁니다
func refreshTokenExpiry(now time.Time, idleTTL time.Duration, familyExpiresAt time.Time) time.Time {
	idleExpiresAt := now.Add(idleTTL)
	if idleExpiresAt.After(familyExpiresAt) {
		return familyExpiresAt
	}
	return idleExpiresAt
}
