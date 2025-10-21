package database

import (
	"database/sql"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	// 테스트 시작 시 .env 파일 로드 (로컬 개발 환경용)
	// 파일이 없어도 에러 무시 (CI/CD 환경에서는 환경변수 직접 설정)
	_ = godotenv.Load("../../.env")
}

// setupTestDB는 테스트용 데이터베이스 연결을 생성합니다
// DATABASE_URL 환경변수가 설정되지 않으면 테스트를 스킵합니다
// 모든 database 패키지의 integration test에서 공통으로 사용됩니다
func setupTestDB(t *testing.T) *sql.DB {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := New(databaseURL)
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	return db
}
