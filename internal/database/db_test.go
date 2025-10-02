package database

import (
	"database/sql"
	"os"
	"testing"
)

// TestNew_EmptyURL_ReturnsError는 빈 DATABASE_URL로 New() 호출 시 에러를 반환하는지 테스트합니다
func TestNew_EmptyURL_ReturnsError(t *testing.T) {
	// Given: 빈 DATABASE_URL
	databaseURL := ""

	// When: New() 호출
	db, err := New(databaseURL)

	// Then: 에러 반환
	if err == nil {
		t.Fatal("expected error for empty DATABASE_URL, got nil")
	}
	if db != nil {
		t.Fatal("expected nil db for empty DATABASE_URL")
	}
}

// TestNew_InvalidURL_ReturnsError는 잘못된 DATABASE_URL로 New() 호출 시 에러를 반환하는지 테스트합니다
func TestNew_InvalidURL_ReturnsError(t *testing.T) {
	// Given: 잘못된 DATABASE_URL
	databaseURL := "invalid://wrong:format@localhost/db"

	// When: New() 호출
	db, err := New(databaseURL)

	// Then: 에러 반환
	if err == nil {
		t.Fatal("expected error for invalid DATABASE_URL, got nil")
	}
	if db != nil {
		t.Fatal("expected nil db for invalid DATABASE_URL")
	}
}

// TestNew_ValidURL_Success는 유효한 DATABASE_URL로 연결 성공하는지 테스트합니다
// 이 테스트는 실제 PostgreSQL 데이터베이스가 필요합니다
func TestNew_ValidURL_Success(t *testing.T) {
	// Given: 환경변수에서 DATABASE_URL 읽기
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	// When: New() 호출
	db, err := New(databaseURL)

	// Then: 연결 성공
	if err != nil {
		t.Fatalf("expected successful connection, got error: %v", err)
	}
	if db == nil {
		t.Fatal("expected non-nil db, got nil")
	}

	// Cleanup
	defer Close(db)

	// Connection Pool 설정 검증
	stats := db.Stats()
	if stats.MaxOpenConnections != 25 {
		t.Errorf("expected MaxOpenConnections=25, got %d", stats.MaxOpenConnections)
	}
}

// TestClose_NilDB_NoError는 nil DB에 대해 Close() 호출 시 에러가 없는지 테스트합니다
func TestClose_NilDB_NoError(t *testing.T) {
	// Given: nil DB
	var db *sql.DB = nil

	// When: Close() 호출
	err := Close(db)

	// Then: 에러 없음
	if err != nil {
		t.Fatalf("expected no error for nil db, got: %v", err)
	}
}

// TestClose_ValidDB_Success는 유효한 DB 연결을 정상적으로 닫는지 테스트합니다
func TestClose_ValidDB_Success(t *testing.T) {
	// Given: 유효한 DB 연결
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := New(databaseURL)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	// When: Close() 호출
	err = Close(db)

	// Then: 에러 없음
	if err != nil {
		t.Fatalf("expected successful close, got error: %v", err)
	}
}
