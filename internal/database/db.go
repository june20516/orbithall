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
	// 단일 인스턴스로 운영하는 저트래픽 API라서 동시에 필요한 연결이 많지 않습니다
	// DB 서버의 최대 연결 수보다 충분히 작게 두어 마이그레이션 도구 등이 함께 접속할 여유를 남깁니다
	db.SetMaxOpenConns(10)
	// 요청이 없을 때 재사용을 위해 풀에 남겨두는 유휴 연결 수
	db.SetMaxIdleConns(5)
	// 연결의 최대 수명 (생성 후 5분이 지나면 재사용하지 않고 닫음)
	// DB 서버가 유휴 연결을 끊거나 스스로 정지하기까지의 시간보다 길지 않게 유지해야
	// 서버 쪽에서 이미 끊긴 연결을 재사용해 쿼리가 실패하는 일을 막을 수 있습니다
	db.SetConnMaxLifetime(5 * time.Minute)
	// 유휴 연결의 최대 유지 시간 (5분 동안 사용되지 않으면 닫음)
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
