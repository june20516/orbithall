package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/testhelpers"
)

func init() {
	// 테스트용 환경변수 설정
	os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long-for-security")
	os.Setenv("JWT_EXPIRATION_HOURS", "168")
	os.Setenv("GOOGLE_CLIENT_ID", "test-client-id.apps.googleusercontent.com")
}

// TestGoogleVerify_MissingFields는 필수 필드 누락 시 400 에러를 테스트합니다
func TestGoogleVerify_MissingFields(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	handler := NewAuthHandler(db)

	tests := []struct {
		name        string
		requestBody map[string]interface{}
	}{
		{
			name:        "id_token 누락",
			requestBody: map[string]interface{}{"email": "test@example.com", "name": "Test User"},
		},
		{
			name:        "email 누락",
			requestBody: map[string]interface{}{"id_token": "token", "name": "Test User"},
		},
		{
			name:        "name 누락",
			requestBody: map[string]interface{}{"id_token": "token", "email": "test@example.com"},
		},
		{
			name:        "빈 요청 본문",
			requestBody: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 필수 필드가 누락된 요청
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/google/verify", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// When: GoogleVerify 호출
			handler.GoogleVerify(rec, req)

			// Then: 400 Bad Request
			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestGoogleVerify_InvalidContentType는 잘못된 Content-Type을 테스트합니다
func TestGoogleVerify_InvalidContentType(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	handler := NewAuthHandler(db)

	// Given: Content-Type이 application/json이 아닌 요청
	req := httptest.NewRequest(http.MethodPost, "/auth/google/verify", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	// When: GoogleVerify 호출
	handler.GoogleVerify(rec, req)

	// Then: 400 Bad Request
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

// TestGoogleVerify_InvalidJSONBody는 잘못된 JSON 형식을 테스트합니다
func TestGoogleVerify_InvalidJSONBody(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	handler := NewAuthHandler(db)

	// Given: 잘못된 JSON
	req := httptest.NewRequest(http.MethodPost, "/auth/google/verify", bytes.NewBuffer([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// When: GoogleVerify 호출
	handler.GoogleVerify(rec, req)

	// Then: 400 Bad Request
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

// TestGoogleVerify_InvalidGoogleToken는 잘못된 Google ID Token을 테스트합니다
func TestGoogleVerify_InvalidGoogleToken(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	handler := NewAuthHandler(db)

	// Given: 잘못된 Google ID Token
	requestBody := map[string]interface{}{
		"id_token": "invalid-google-token",
		"email":    "test@example.com",
		"name":     "Test User",
		"picture":  "https://example.com/pic.jpg",
	}
	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/google/verify", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// When: GoogleVerify 호출
	handler.GoogleVerify(rec, req)

	// Then: 401 Unauthorized (Google 검증 실패)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

// 참고: 실제 Google ID Token 검증 및 JWT 발급 테스트는 통합 테스트에서 수행
// 실제 토큰을 사용하려면 Google OAuth Playground에서 발급받아야 함
// 또는 VerifyGoogleIDToken을 모킹하는 별도 테스트가 필요함
