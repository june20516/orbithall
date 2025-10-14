package main

import (
	"os"
	"testing"
)

// TestRun_MissingDatabaseURL_ReturnsError는 DATABASE_URL이 없을 때 에러를 반환하는지 테스트합니다
func TestRun_MissingDatabaseURL_ReturnsError(t *testing.T) {
	// Given: DATABASE_URL 환경변수 제거
	originalURL, wasSet := os.LookupEnv("DATABASE_URL")
	os.Unsetenv("DATABASE_URL")
	defer func() {
		if wasSet {
			os.Setenv("DATABASE_URL", originalURL)
		}
	}()

	// When: run() 호출
	err := run()

	// Then: 에러 반환
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is missing, got nil")
	}

	expectedMsg := "DATABASE_URL environment variable is required"
	if err.Error() != expectedMsg {
		t.Errorf("expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestRun_InvalidDatabaseURL_ReturnsError는 잘못된 DATABASE_URL일 때 에러를 반환하는지 테스트합니다
func TestRun_InvalidDatabaseURL_ReturnsError(t *testing.T) {
	// Given: 잘못된 DATABASE_URL
	originalURL, wasSet := os.LookupEnv("DATABASE_URL")
	os.Setenv("DATABASE_URL", "invalid://wrong")
	defer func() {
		if wasSet {
			os.Setenv("DATABASE_URL", originalURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}()

	// When: run() 호출
	err := run()

	// Then: 데이터베이스 연결 실패 에러 반환
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is invalid, got nil")
	}

	t.Logf("expected behavior: database connection failed - %v", err)
}
