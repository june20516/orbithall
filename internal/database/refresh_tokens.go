package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/june20516/orbithall/internal/models"
)

// refreshTokenColumns는 refresh_tokens 조회 시 공통으로 읽는 컬럼 목록입니다
// scanRefreshToken의 Scan 순서와 같아야 합니다
const refreshTokenColumns = `id, user_id, family_id, parent_id, token_hash, expires_at,
	family_expires_at, used_at, revoked_at, revoked_reason, created_at`

// scanRefreshToken은 refreshTokenColumns 순서의 한 행을 RefreshToken으로 읽습니다
func scanRefreshToken(row *sql.Row) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.FamilyID,
		&token.ParentID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.FamilyExpiresAt,
		&token.UsedAt,
		&token.RevokedAt,
		&token.RevokedReason,
		&token.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// CreateRefreshTokenFamily는 로그인 시 새 계열의 첫 Refresh Token을 저장합니다
// family_id는 DB가 gen_random_uuid()로 만듭니다
func CreateRefreshTokenFamily(ctx context.Context, db DBTX, userID int64, tokenHash []byte, expiresAt, familyExpiresAt time.Time) (*models.RefreshToken, error) {
	query := `
		INSERT INTO refresh_tokens (user_id, family_id, token_hash, expires_at, family_expires_at)
		VALUES ($1, gen_random_uuid(), $2, $3, $4)
		RETURNING ` + refreshTokenColumns

	token, err := scanRefreshToken(db.QueryRowContext(ctx, query, userID, tokenHash, expiresAt, familyExpiresAt))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return token, nil
}

// GetRefreshTokenByHash는 토큰 해시로 Refresh Token을 조회합니다
// 찾지 못한 경우 nil을 반환합니다
func GetRefreshTokenByHash(ctx context.Context, db DBTX, tokenHash []byte) (*models.RefreshToken, error) {
	query := `SELECT ` + refreshTokenColumns + ` FROM refresh_tokens WHERE token_hash = $1`

	token, err := scanRefreshToken(db.QueryRowContext(ctx, query, tokenHash))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token by hash: %w", err)
	}

	return token, nil
}

// GetChildRefreshToken은 parentID 토큰을 회전해 만든 후속 토큰을 조회합니다
// 아직 회전하지 않은 토큰이면 nil을 반환합니다
func GetChildRefreshToken(ctx context.Context, db DBTX, parentID int64) (*models.RefreshToken, error) {
	query := `SELECT ` + refreshTokenColumns + ` FROM refresh_tokens WHERE parent_id = $1`

	token, err := scanRefreshToken(db.QueryRowContext(ctx, query, parentID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get child refresh token: %w", err)
	}

	return token, nil
}

// RotateRefreshToken은 parentID 토큰을 사용 처리하고 같은 계열의 후속 토큰을 저장합니다
//
// 부모가 아직 사용되지 않았고 폐기되지 않은 경우에만 두 작업을 한 SQL 문장으로 처리합니다
// 같은 부모로 동시에 요청이 와도 한 요청만 성공하고, 나머지는 nil을 받습니다
// (부모 행을 먼저 잠근 요청이 커밋되면 다른 요청의 WHERE 조건이 다시 평가되어 0행이 됩니다)
//
// 같은 계열에 폐기된 토큰이 하나라도 있으면 회전하지 않습니다
// 계열 폐기 UPDATE는 문장 시작 시점에 보이는 행만 폐기하므로, 동시에 진행 중이던 회전이 만든 후속 토큰은
// 폐기되지 않은 채 남을 수 있습니다. 이 조건이 그런 토큰으로 세션이 이어지는 것을 막습니다
//
// 부모가 이미 사용되었거나, 폐기되었거나, 계열이 폐기되었으면 nil, nil을 반환합니다
func RotateRefreshToken(ctx context.Context, db DBTX, parentID int64, childHash []byte, childExpiresAt, now time.Time) (*models.RefreshToken, error) {
	// INSERT ... SELECT의 SELECT 목록에 쓴 파라미터는 타입을 추론하지 못하므로 명시적으로 캐스팅합니다
	query := `
		WITH parent AS (
			UPDATE refresh_tokens
			SET used_at = $2
			WHERE id = $1 AND used_at IS NULL AND revoked_at IS NULL
				AND NOT EXISTS (
					SELECT 1 FROM refresh_tokens AS revoked
					WHERE revoked.family_id = refresh_tokens.family_id AND revoked.revoked_at IS NOT NULL
				)
			RETURNING id, user_id, family_id, family_expires_at
		)
		INSERT INTO refresh_tokens (user_id, family_id, parent_id, token_hash, expires_at, family_expires_at)
		SELECT user_id, family_id, id, $3::bytea, $4::timestamptz, family_expires_at
		FROM parent
		RETURNING ` + refreshTokenColumns

	token, err := scanRefreshToken(db.QueryRowContext(ctx, query, parentID, now, childHash, childExpiresAt))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to rotate refresh token: %w", err)
	}

	return token, nil
}

// RevokeRefreshTokenFamily는 계열의 모든 토큰을 폐기합니다
// 이미 폐기된 토큰은 처음 폐기 정보를 유지합니다 (여러 번 호출해도 결과가 같음)
func RevokeRefreshTokenFamily(ctx context.Context, db DBTX, familyID string, reason string, now time.Time) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $2, revoked_reason = $3
		WHERE family_id = $1 AND revoked_at IS NULL
	`

	if _, err := db.ExecContext(ctx, query, familyID, now, reason); err != nil {
		return fmt.Errorf("failed to revoke refresh token family: %w", err)
	}

	return nil
}

// DeleteStaleRefreshTokens는 사용자의 토큰 중 before 이전에 절대 만료되었거나 폐기된 행을 삭제합니다
// 로그인 시 호출해 별도 스케줄러 없이 테이블이 계속 커지지 않게 합니다
func DeleteStaleRefreshTokens(ctx context.Context, db DBTX, userID int64, before time.Time) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE user_id = $1 AND (family_expires_at < $2 OR revoked_at < $2)
	`

	if _, err := db.ExecContext(ctx, query, userID, before); err != nil {
		return fmt.Errorf("failed to delete stale refresh tokens: %w", err)
	}

	return nil
}
