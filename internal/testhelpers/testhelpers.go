package testhelpers

import (
	"database/sql"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/june20516/orbithall/internal/database"
	"github.com/lib/pq"
)

func init() {
	// 테스트 시작 시 .env 파일 로드 (로컬 개발 환경용)
	// 파일이 없어도 에러 무시 (CI/CD 환경에서는 환경변수 직접 설정)
	_ = godotenv.Load("../../.env")
}

// SetupTestDB는 테스트용 데이터베이스 연결을 생성합니다
// DATABASE_URL 환경변수가 설정되지 않으면 테스트를 스킵합니다
// 모든 integration test에서 공통으로 사용됩니다
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := database.New(databaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	return db
}

// CleanupSites는 sites 테이블의 모든 데이터를 삭제합니다
// CASCADE 옵션으로 관련된 posts, comments도 함께 삭제됩니다
func CleanupSites(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("TRUNCATE sites RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to cleanup sites: %v", err)
	}
}

// CleanupPosts는 posts 테이블의 모든 데이터를 삭제합니다
// CASCADE 옵션으로 관련된 comments도 함께 삭제됩니다
func CleanupPosts(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("TRUNCATE posts RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to cleanup posts: %v", err)
	}
}

// CleanupComments는 comments 테이블의 모든 데이터를 삭제합니다
func CleanupComments(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("TRUNCATE comments RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to cleanup comments: %v", err)
	}
}

// CreateTestSite는 테스트용 사이트를 생성하고 API 키를 반환합니다
// 사이트 생성 후 자동으로 생성된 UUID API 키를 반환합니다
func CreateTestSite(t *testing.T, db *sql.DB, name string, domain string, corsOrigins []string, isActive bool) string {
	t.Helper()

	query := `
		INSERT INTO sites (name, domain, cors_origins, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING api_key
	`

	var apiKey string
	err := db.QueryRow(query, name, domain, pq.StringArray(corsOrigins), isActive).Scan(&apiKey)
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	return apiKey
}

// CreateTestPost는 테스트용 포스트를 생성하고 포스트 ID를 반환합니다
func CreateTestPost(t *testing.T, db *sql.DB, siteID int64, slug string, title string) int64 {
	t.Helper()

	query := `
		INSERT INTO posts (site_id, slug, title, comment_count)
		VALUES ($1, $2, $3, 0)
		RETURNING id
	`

	var postID int64
	err := db.QueryRow(query, siteID, slug, title).Scan(&postID)
	if err != nil {
		t.Fatalf("Failed to create test post: %v", err)
	}

	return postID
}

// CreateTestSiteWithID는 특정 ID로 테스트용 사이트를 생성합니다
// 외래 키 제약조건 때문에 특정 site_id가 필요한 경우 사용합니다
func CreateTestSiteWithID(t *testing.T, db *sql.DB, id int64, name string, domain string, corsOrigins []string, isActive bool) {
	t.Helper()

	query := `
		INSERT INTO sites (id, name, domain, cors_origins, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := db.Exec(query, id, name, domain, pq.StringArray(corsOrigins), isActive)
	if err != nil {
		t.Fatalf("Failed to create test site with ID: %v", err)
	}
}
