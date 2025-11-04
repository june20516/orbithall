package database

import (
	"context"
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
