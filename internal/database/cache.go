package database

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/june20516/orbithall/internal/models"
	"github.com/lib/pq"
)

// cacheEntry는 캐시된 사이트 정보와 만료 시간을 저장합니다
type cacheEntry struct {
	site      *models.Site
	expiresAt time.Time
}

// isExpired는 캐시 항목이 만료되었는지 확인합니다
func (e *cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// SiteCache는 사이트 정보를 캐싱하는 thread-safe 캐시입니다
var siteCache sync.Map

// cacheTTL은 캐시 항목의 유효 시간입니다 (1분)
const cacheTTL = 1 * time.Minute

// GetSiteByAPIKey는 API 키로 사이트 정보를 조회합니다
// 캐시에 있고 만료되지 않았으면 캐시에서 반환하고, 없거나 만료되었으면 DB에서 조회합니다
func GetSiteByAPIKey(db *sql.DB, apiKey string) (*models.Site, error) {
	// 캐시 조회
	if cached, ok := siteCache.Load(apiKey); ok {
		entry := cached.(*cacheEntry)
		// TTL 확인
		if !entry.isExpired() {
			return entry.site, nil
		}
		// 만료된 항목 삭제
		siteCache.Delete(apiKey)
	}

	// 캐시 미스: DB에서 조회
	site, err := getSiteFromDB(db, apiKey)
	if err != nil {
		return nil, err
	}

	// 캐시에 저장
	siteCache.Store(apiKey, &cacheEntry{
		site:      site,
		expiresAt: time.Now().Add(cacheTTL),
	})

	return site, nil
}

// getSiteFromDB는 데이터베이스에서 API 키로 사이트 정보를 조회합니다
func getSiteFromDB(db *sql.DB, apiKey string) (*models.Site, error) {
	query := `
		SELECT id, name, domain, api_key, cors_origins, is_active, created_at, updated_at
		FROM sites
		WHERE api_key = $1 AND is_active = true
	`

	var site models.Site
	var corsOrigins pq.StringArray

	err := db.QueryRow(query, apiKey).Scan(
		&site.ID,
		&site.Name,
		&site.Domain,
		&site.APIKey,
		&corsOrigins,
		&site.IsActive,
		&site.CreatedAt,
		&site.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("site not found or inactive")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query site: %w", err)
	}

	// PostgreSQL TEXT[] 타입을 []string으로 변환
	site.CORSOrigins = []string(corsOrigins)

	return &site, nil
}
