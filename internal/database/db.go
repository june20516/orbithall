package database

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
func New(databaseURL string) (*sql.DB, error) {
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
	// Supabase Session Pooler는 최대 15개 연결 제한
	// 로컬 개발은 더 높은 값 사용 가능
	db.SetMaxOpenConns(10)  // 안전한 기본값 (Session Pooler 15개 제한 고려)
	db.SetMaxIdleConns(5)   // 유휴 연결은 적게 유지
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
