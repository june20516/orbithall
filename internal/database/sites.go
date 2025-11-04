package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/june20516/orbithall/internal/models"
	"github.com/lib/pq"
)

// CreateSiteForUser는 사이트를 생성하고 사용자를 owner로 연결합니다
// 트랜잭션 내에서 두 작업을 수행하여 원자성을 보장합니다
// API Key는 자동으로 생성됩니다 (orb_live_ prefix)
func CreateSiteForUser(ctx context.Context, db DBTX, site *models.Site, userID int64) error {
	// API Key 자동 생성
	if site.APIKey == "" {
		site.APIKey = models.GenerateAPIKey("orb_live_")
	}

	// 사이트 생성
	query := `
		INSERT INTO sites (name, domain, api_key, cors_origins, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := db.QueryRowContext(ctx, query,
		site.Name,
		site.Domain,
		site.APIKey,
		pq.Array(site.CORSOrigins),
		site.IsActive,
	).Scan(&site.ID, &site.CreatedAt, &site.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create site: %w", err)
	}

	// 사용자를 사이트에 owner로 연결
	if err := AddUserToSite(ctx, db, userID, site.ID, "owner"); err != nil {
		return fmt.Errorf("failed to add user to site: %w", err)
	}

	return nil
}

// GetSiteByID는 ID로 사이트를 조회합니다
// 사이트가 존재하지 않으면 sql.ErrNoRows를 반환합니다
func GetSiteByID(ctx context.Context, db DBTX, siteID int64) (*models.Site, error) {
	query := `
		SELECT id, name, domain, api_key, cors_origins, is_active, created_at, updated_at
		FROM sites
		WHERE id = $1
	`

	site := &models.Site{}
	err := db.QueryRowContext(ctx, query, siteID).Scan(
		&site.ID,
		&site.Name,
		&site.Domain,
		&site.APIKey,
		pq.Array(&site.CORSOrigins),
		&site.IsActive,
		&site.CreatedAt,
		&site.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	return site, nil
}

// UpdateSite는 사이트 정보를 수정합니다
// name, cors_origins, is_active 필드만 수정 가능합니다
// domain과 api_key는 수정 불가능합니다
func UpdateSite(ctx context.Context, db DBTX, siteID int64, name string, corsOrigins []string, isActive bool) error {
	query := `
		UPDATE sites
		SET name = $1, cors_origins = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
	`

	result, err := db.ExecContext(ctx, query,
		name,
		pq.Array(corsOrigins),
		isActive,
		siteID,
	)

	if err != nil {
		return fmt.Errorf("failed to update site: %w", err)
	}

	// 영향받은 행 수 확인
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// DeleteSite는 사이트를 삭제합니다
// CASCADE 설정으로 인해 연결된 posts, comments, user_sites도 자동 삭제됩니다
func DeleteSite(ctx context.Context, db DBTX, siteID int64) error {
	query := `
		DELETE FROM sites
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, siteID)
	if err != nil {
		return fmt.Errorf("failed to delete site: %w", err)
	}

	// 영향받은 행 수 확인
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
