package database

import (
	"context"
	"fmt"

	"github.com/june20516/orbithall/internal/models"
	"github.com/lib/pq"
)

// AddUserToSite는 사용자를 사이트에 연결합니다
// role 기본값: "owner"
// 중복 연결 시 복합 PK 위반 에러 반환
func AddUserToSite(ctx context.Context, db DBTX, userID, siteID int64, role string) error {
	query := `
		INSERT INTO user_sites (user_id, site_id, role)
		VALUES ($1, $2, $3)
	`

	_, err := db.ExecContext(ctx, query, userID, siteID, role)
	if err != nil {
		return fmt.Errorf("failed to add user to site: %w", err)
	}

	return nil
}

// GetUserSites는 사용자가 소유한 사이트 목록을 조회합니다
// JOIN 쿼리로 N+1 문제 방지
// created_at 내림차순 정렬 (최신 사이트 먼저)
func GetUserSites(ctx context.Context, db DBTX, userID int64) ([]models.Site, error) {
	query := `
		SELECT
			s.id, s.name, s.domain, s.api_key, s.cors_origins, s.is_active,
			s.created_at, s.updated_at
		FROM sites s
		INNER JOIN user_sites us ON s.id = us.site_id
		WHERE us.user_id = $1
		ORDER BY s.created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sites: %w", err)
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var site models.Site
		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan site: %w", err)
		}
		sites = append(sites, site)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating site rows: %w", err)
	}

	// 빈 배열 반환 (nil 아님)
	if sites == nil {
		sites = []models.Site{}
	}

	return sites, nil
}

// GetSiteUsers는 사이트에 연결된 사용자 목록을 조회합니다
// JOIN 쿼리로 N+1 문제 방지
func GetSiteUsers(ctx context.Context, db DBTX, siteID int64) ([]models.User, error) {
	query := `
		SELECT
			u.id, u.email, u.name, u.picture_url, u.google_id,
			u.created_at, u.updated_at
		FROM users u
		INNER JOIN user_sites us ON u.id = us.user_id
		WHERE us.site_id = $1
		ORDER BY us.created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to query site users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.PictureURL,
			&user.GoogleID,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	// 빈 배열 반환 (nil 아님)
	if users == nil {
		users = []models.User{}
	}

	return users, nil
}

// RemoveUserFromSite는 사용자-사이트 연결을 해제합니다
func RemoveUserFromSite(ctx context.Context, db DBTX, userID, siteID int64) error {
	query := `
		DELETE FROM user_sites
		WHERE user_id = $1 AND site_id = $2
	`

	_, err := db.ExecContext(ctx, query, userID, siteID)
	if err != nil {
		return fmt.Errorf("failed to remove user from site: %w", err)
	}

	return nil
}

// HasUserSiteAccess는 사용자가 사이트에 접근 권한이 있는지 확인합니다
// role = 'owner'인 경우에만 true 반환
func HasUserSiteAccess(ctx context.Context, db DBTX, userID, siteID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_sites
			WHERE user_id = $1 AND site_id = $2 AND role = 'owner'
		)
	`

	var exists bool
	err := db.QueryRowContext(ctx, query, userID, siteID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check site ownership: %w", err)
	}

	return exists, nil
}
