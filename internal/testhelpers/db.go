package testhelpers

/** 의존성 순환참조를 해결하기 위한 중복 코드 */
import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// New는 PostgreSQL 데이터베이스 연결을 생성하고 Connection Pool을 설정합니다
// databaseURL 형식: postgres://user:password@host:port/dbname?sslmode=disable
func NewDB(databaseURL string) (*sql.DB, error) {
	// DATABASE_URL 검증
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	// 데이터베이스 연결 열기
	// sql.Open()은 실제로 연결하지 않고 DB 객체만 생성합니다
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection Pool 설정
	// 동시에 열 수 있는 최대 연결 수
	db.SetMaxOpenConns(100)
	// 유휴 상태로 유지할 최대 연결 수
	db.SetMaxIdleConns(100)
	// 연결의 최대 수명 (5분 후 자동 닫힘)
	db.SetConnMaxLifetime(5 * time.Minute)
	// 유휴 연결의 최대 유지 시간 (5분 후 자동 닫힘)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// 실제 데이터베이스 연결 테스트
	// Ping()을 호출해야 실제로 연결이 시도됩니다
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// Close는 데이터베이스 연결을 종료합니다
// 모든 활성 연결과 유휴 연결을 정리합니다
func Close(db *sql.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}
