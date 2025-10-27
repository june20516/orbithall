package testhelpers

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"github.com/june20516/orbithall/internal/models"
	"github.com/lib/pq"
)

func init() {
	// 테스트 시작 시 .env 파일 로드 (로컬 개발 환경용)
	// 파일이 없어도 에러 무시 (CI/CD 환경에서는 환경변수 직접 설정)
	_ = godotenv.Load("../../.env")
}

// SetupTestDB는 테스트용 데이터베이스 연결을 생성하고 마이그레이션을 실행합니다
// TEST_DATABASE_URL 환경변수가 설정되지 않으면 테스트를 스킵합니다
// 모든 integration test에서 공통으로 사용됩니다
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	db, err := NewDB(databaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 마이그레이션 자동 실행
	if err := runMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// runMigrations는 golang-migrate를 사용하여 테스트 DB에 마이그레이션을 실행합니다
func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://../../migrations",
		"postgres", driver)
	if err != nil {
		return err
	}

	// 마이그레이션 실행 (이미 최신 상태면 에러 무시)
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func SetupTxTest(t *testing.T, db *sql.DB) (context.Context, DBTX, func()) {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// 롤백 함수
	cleanup := func() {
		tx.Rollback()
	}

	return context.Background(), tx, cleanup
}

// CreateTestSite는 테스트용 사이트를 생성하고 API 키를 반환합니다
// 테스트용 API 키는 "orb_test_" prefix를 사용합니다
func CreateTestSite(ctx context.Context, t *testing.T, db DBTX, name string, domain string, corsOrigins []string, isActive bool) models.Site {
	t.Helper()

	// 테스트용 API 키 생성
	apiKey := models.GenerateAPIKey("orb_test_")

	query := `
		INSERT INTO sites (name, domain, api_key, cors_origins, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, domain, api_key, cors_origins, is_active, created_at, updated_at
	`

	var site models.Site
	err := db.QueryRowContext(ctx, query, name, domain, apiKey, pq.StringArray(corsOrigins), isActive).Scan(
		&site.ID, &site.Name, &site.Domain, &site.APIKey, pq.Array(&site.CORSOrigins), &site.IsActive, &site.CreatedAt, &site.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	return site
}

// CreateTestPost는 테스트용 포스트를 생성하고 포스트 ID를 반환합니다
func CreateTestPost(ctx context.Context, t *testing.T, db DBTX, siteID int64, slug string, title string) models.Post {
	t.Helper()

	query := `
		INSERT INTO posts (site_id, slug, title, comment_count)
		VALUES ($1, $2, $3, 0)
		RETURNING id, site_id, slug, title, comment_count, created_at, updated_at
	`

	var post models.Post
	err := db.QueryRowContext(ctx, query, siteID, slug, title).Scan(
		&post.ID, &post.SiteID, &post.Slug, &post.Title, &post.CommentCount, &post.CreatedAt, &post.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("Failed to create test post: %v", err)
	}

	return post
}

// CreateTestSiteWithID는 특정 ID로 테스트용 사이트를 생성합니다
// 외래 키 제약조건 때문에 특정 site_id가 필요한 경우 사용합니다
func CreateTestSiteWithID(ctx context.Context, t *testing.T, db DBTX, id int64, name string, domain string, corsOrigins []string, isActive bool) {
	t.Helper()

	// 테스트용 API 키 생성
	apiKey := models.GenerateAPIKey("orb_test_")

	query := `
		INSERT INTO sites (id, name, domain, api_key, cors_origins, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := db.ExecContext(ctx, query, id, name, domain, apiKey, pq.StringArray(corsOrigins), isActive)
	if err != nil {
		t.Fatalf("Failed to create test site with ID: %v", err)
	}
}
