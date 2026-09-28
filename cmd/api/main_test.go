package main

import (
	"os"
	"strings"
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
	// Given: 검증을 통과하는 인증 비밀키와 잘못된 DATABASE_URL
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-characters-long")
	t.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-secret-at-least-32-characters")
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

// TestRun_InvalidRefreshTokenSecret_ReturnsError는 REFRESH_TOKEN_SECRET이 잘못되면 서버가 시작되지 않는지 테스트합니다
func TestRun_InvalidRefreshTokenSecret_ReturnsError(t *testing.T) {
	// Given: 연결을 시도하면 실패할 DATABASE_URL과 비어 있는 비밀키
	// 비밀키 검증이 DB 연결보다 먼저이므로 DB 없이도 비밀키 에러가 나야 합니다
	t.Setenv("DATABASE_URL", "postgres://unused")
	t.Setenv("REFRESH_TOKEN_SECRET", "")

	// When: run() 호출
	err := run()

	// Then: 비밀키 에러로 시작 실패
	if err == nil {
		t.Fatal("expected error when REFRESH_TOKEN_SECRET is invalid, got nil")
	}
	if !strings.Contains(err.Error(), "REFRESH_TOKEN_SECRET") {
		t.Errorf("expected REFRESH_TOKEN_SECRET error, got: %v", err)
	}
}
