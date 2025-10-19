package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/testhelpers"
)

// ============================================
// 테스트 케이스
// ============================================

func TestAuthMiddleware_MissingAPIKey(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// 더미 핸들러 (인증 성공 시 호출됨)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	// AuthMiddleware 적용
	handler := AuthMiddleware(db)(nextHandler)

	// 테스트 요청 생성 (API 키 헤더 없음)
	req := httptest.NewRequest(http.MethodGet, "/api/comments", nil)
	rec := httptest.NewRecorder()

	// 요청 실행
	handler.ServeHTTP(rec, req)

	// 검증: 401 Unauthorized
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	// 응답 본문 검증
	var response ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error.Code != ErrMissingAPIKey {
		t.Errorf("Expected error code %s, got %s", ErrMissingAPIKey, response.Error.Code)
	}

	if response.Error.Message != "API key is required" {
		t.Errorf("Expected message 'API key is required', got %s", response.Error.Message)
	}
}

func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// 더미 핸들러
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	// AuthMiddleware 적용
	handler := AuthMiddleware(db)(nextHandler)

	// 테스트 요청 생성 (잘못된 API 키)
	req := httptest.NewRequest(http.MethodGet, "/api/comments", nil)
	req.Header.Set("X-Orbithall-API-Key", "invalid-key-12345")
	rec := httptest.NewRecorder()

	// 요청 실행
	handler.ServeHTTP(rec, req)

	// 검증: 403 Forbidden
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	// 응답 본문 검증
	var response ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error.Code != ErrInvalidAPIKey {
		t.Errorf("Expected error code %s, got %s", ErrInvalidAPIKey, response.Error.Code)
	}
}

func TestAuthMiddleware_InactiveSite(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// 비활성 사이트 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Inactive Site", "inactive.com", []string{"http://localhost:3000"}, false)

	// 더미 핸들러
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	// AuthMiddleware 적용
	handler := AuthMiddleware(db)(nextHandler)

	// 테스트 요청 생성
	req := httptest.NewRequest(http.MethodGet, "/api/comments", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)
	rec := httptest.NewRecorder()

	// 요청 실행
	handler.ServeHTTP(rec, req)

	// 검증: 403 Forbidden (GetSiteByAPIKey가 is_active=false는 반환하지 않음)
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	// 응답 본문 검증
	var response ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error.Code != ErrInvalidAPIKey {
		t.Errorf("Expected error code %s, got %s", ErrInvalidAPIKey, response.Error.Code)
	}
}

func TestAuthMiddleware_InvalidOrigin(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// 활성 사이트 생성 (CORS: http://localhost:3000만 허용)
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "test.com", []string{"http://localhost:3000"}, true)

	// 더미 핸들러
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	// AuthMiddleware 적용
	handler := AuthMiddleware(db)(nextHandler)

	// 테스트 요청 생성 (허용되지 않은 Origin)
	req := httptest.NewRequest(http.MethodGet, "/api/comments", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)
	req.Header.Set("Origin", "http://malicious-site.com")
	rec := httptest.NewRecorder()

	// 요청 실행
	handler.ServeHTTP(rec, req)

	// 검증: 403 Forbidden
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	// 응답 본문 검증
	var response ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Error.Code != ErrInvalidOrigin {
		t.Errorf("Expected error code %s, got %s", ErrInvalidOrigin, response.Error.Code)
	}
}

func TestAuthMiddleware_ValidAPIKeyAndOrigin(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// 활성 사이트 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "test.com", []string{"http://localhost:3000"}, true)

	// 더미 핸들러 (Context에서 사이트 정보 추출)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Context에서 사이트 정보 확인
		site := GetSiteFromContext(r.Context())
		if site == nil {
			t.Error("Expected site in context, got nil")
		}
		if site.Name != "Test Site" {
			t.Errorf("Expected site name 'Test Site', got %s", site.Name)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	// AuthMiddleware 적용
	handler := AuthMiddleware(db)(nextHandler)

	// 테스트 요청 생성
	req := httptest.NewRequest(http.MethodGet, "/api/comments", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()

	// 요청 실행
	handler.ServeHTTP(rec, req)

	// 검증: 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthMiddleware_NoOriginHeader(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// 활성 사이트 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "test.com", []string{"http://localhost:3000"}, true)

	// 더미 핸들러
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	// AuthMiddleware 적용
	handler := AuthMiddleware(db)(nextHandler)

	// 테스트 요청 생성 (Origin 헤더 없음 - 서버 간 요청 시뮬레이션)
	req := httptest.NewRequest(http.MethodGet, "/api/comments", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)
	// Origin 헤더를 설정하지 않음
	rec := httptest.NewRecorder()

	// 요청 실행
	handler.ServeHTTP(rec, req)

	// 검증: 200 OK (Origin 헤더가 없으면 CORS 검증 스킵)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthMiddleware_CaseInsensitiveOrigin(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// 활성 사이트 생성 (소문자 CORS)
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "test.com", []string{"http://localhost:3000"}, true)

	// 더미 핸들러
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	})

	// AuthMiddleware 적용
	handler := AuthMiddleware(db)(nextHandler)

	// 테스트 요청 생성 (대문자 Origin)
	req := httptest.NewRequest(http.MethodGet, "/api/comments", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)
	req.Header.Set("Origin", "http://LOCALHOST:3000") // 대문자
	rec := httptest.NewRecorder()

	// 요청 실행
	handler.ServeHTTP(rec, req)

	// 검증: 200 OK (대소문자 무시하고 매칭)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// ============================================
// 헬퍼 함수 테스트
// ============================================

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		allowedOrigins []string
		expected       bool
	}{
		{
			name:           "Exact match",
			origin:         "http://localhost:3000",
			allowedOrigins: []string{"http://localhost:3000", "https://example.com"},
			expected:       true,
		},
		{
			name:           "Case insensitive match",
			origin:         "http://LOCALHOST:3000",
			allowedOrigins: []string{"http://localhost:3000"},
			expected:       true,
		},
		{
			name:           "No match",
			origin:         "http://malicious.com",
			allowedOrigins: []string{"http://localhost:3000"},
			expected:       false,
		},
		{
			name:           "Empty allowed origins",
			origin:         "http://localhost:3000",
			allowedOrigins: []string{},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOriginAllowed(tt.origin, tt.allowedOrigins)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
