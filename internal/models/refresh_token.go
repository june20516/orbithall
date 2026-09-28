package models

import "time"

// Refresh Token 폐기 사유
const (
	// RefreshTokenRevokedByLogout은 사용자가 로그아웃해 폐기된 경우입니다
	RefreshTokenRevokedByLogout = "logout"

	// RefreshTokenRevokedByReuse는 이미 사용된 토큰이 다시 제출되어 탈취로 판단한 경우입니다
	RefreshTokenRevokedByReuse = "reuse_detected"
)

// RefreshToken은 어드민 로그인 세션을 연장하는 Refresh Token의 저장 정보입니다
// 토큰 원문은 저장하지 않고 SHA-256 해시만 보관합니다
// 서버 내부에서만 쓰며 API 응답으로 내보내지 않습니다
type RefreshToken struct {
	// UserID는 토큰을 소유한 사용자 ID입니다
	UserID int64

	// FamilyID는 한 번의 로그인에서 회전으로 이어진 토큰들이 공유하는 식별자(UUID)입니다
	// 재사용이 탐지되면 같은 FamilyID의 토큰을 모두 폐기합니다
	FamilyID string

	// ParentID는 이 토큰을 만들 때 사용 처리된 이전 토큰의 ID입니다
	// 로그인 시 처음 발급된 토큰은 nil입니다
	ParentID *int64

	// TokenHash는 토큰 원문의 SHA-256 해시입니다
	TokenHash []byte

	// ExpiresAt은 이 토큰의 만료 시각입니다
	// 유휴 만료(발급 시각 + 유휴 수명)와 FamilyExpiresAt 중 이른 시각입니다
	ExpiresAt time.Time

	// FamilyExpiresAt은 계열 전체의 절대 만료 시각입니다 (최초 로그인 시각 + 절대 수명)
	FamilyExpiresAt time.Time

	// UsedAt은 이 토큰으로 회전한 시각입니다 (아직 사용 전이면 nil)
	UsedAt *time.Time

	// RevokedAt은 폐기 시각입니다 (폐기 전이면 nil)
	RevokedAt *time.Time

	// RevokedReason은 폐기 사유입니다 (RefreshTokenRevokedBy* 상수 중 하나)
	RevokedReason *string

	// 메타데이터
	ID        int64
	CreatedAt time.Time
}

// IsExpired는 now 시점에 토큰이 만료되었는지 반환합니다
// ExpiresAt은 항상 FamilyExpiresAt 이하로 저장되므로 ExpiresAt만 비교합니다
func (t *RefreshToken) IsExpired(now time.Time) bool {
	return !now.Before(t.ExpiresAt)
}
