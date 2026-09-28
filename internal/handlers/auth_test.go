package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

func init() {
	// 테스트용 환경변수 설정
	os.Setenv("JWT_SECRET", "test-secret-key-at-least-32-characters-long-for-security")
	os.Setenv("JWT_EXPIRATION_HOURS", "168")
	os.Setenv("GOOGLE_CLIENT_ID", "test-client-id.apps.googleusercontent.com")
}

// TestGoogleVerify_MissingFields는 필수 필드(id_token) 누락 시 400 에러를 테스트합니다
// email·name·picture는 선택이므로 누락되어도 400이 아닙니다 (TestGoogleVerify_UsesVerifiedClaims 참고)
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
			if code := readErrorCode(t, rec); code != ErrInvalidInput {
				t.Errorf("error.code = %q, want %q", code, ErrInvalidInput)
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
	if code := readErrorCode(t, rec); code != ErrInvalidInput {
		t.Errorf("error.code = %q, want %q", code, ErrInvalidInput)
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
	if code := readErrorCode(t, rec); code != ErrInvalidInput {
		t.Errorf("error.code = %q, want %q", code, ErrInvalidInput)
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
	if code := readErrorCode(t, rec); code != ErrInvalidIDToken {
		t.Errorf("error.code = %q, want %q", code, ErrInvalidIDToken)
	}
}

// googleVerifyTestTime은 로그인 성공 경로 테스트에 주입하는 고정 시각입니다
var googleVerifyTestTime = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// newGoogleVerifyTestIdentity는 커밋되는 테스트 데이터용 고유 google_id와 email을 만들고,
// 테스트 종료 시 그 google_id의 사용자를 삭제합니다 (CASCADE로 토큰도 함께 삭제됨)
func newGoogleVerifyTestIdentity(t *testing.T, db *sql.DB) (googleID string, email string) {
	t.Helper()
	suffix := fmt.Sprintf("google-verify-%d", time.Now().UnixNano())
	googleID = "google-" + suffix
	email = suffix + "@example.com"
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DELETE FROM users WHERE google_id = $1`, googleID); err != nil {
			t.Errorf("failed to delete committed test user: %v", err)
		}
	})
	return googleID, email
}

// newGoogleVerifyTestHandler는 Google 검증 결과를 고정 payload로, 현재 시각을 고정 시각으로 바꾼 핸들러를 만듭니다
func newGoogleVerifyTestHandler(db *sql.DB, googleID string, email string) *AuthHandler {
	return newGoogleVerifyTestHandlerWithPayload(db, auth.GoogleIDTokenPayload{GoogleID: googleID, Email: email, Name: "Google Verify User"})
}

// newGoogleVerifyTestHandlerWithPayload는 Google 검증 결과로 payload를 돌려주는 핸들러를 만듭니다
func newGoogleVerifyTestHandlerWithPayload(db *sql.DB, payload auth.GoogleIDTokenPayload) *AuthHandler {
	handler := NewAuthHandler(db)
	handler.now = func() time.Time { return googleVerifyTestTime }
	handler.verifyIDToken = func(ctx context.Context, idToken string) (*auth.GoogleIDTokenPayload, error) {
		verified := payload
		return &verified, nil
	}
	return handler
}

// callGoogleVerify는 유효한 형식의 로그인 요청으로 GoogleVerify를 호출합니다
func callGoogleVerify(t *testing.T, handler *AuthHandler, email string) *httptest.ResponseRecorder {
	t.Helper()
	return callGoogleVerifyWithBody(t, handler, map[string]interface{}{
		"id_token": "fake-google-id-token",
		"email":    email,
		"name":     "Google Verify User",
	})
}

// callGoogleVerifyWithBody는 주어진 요청 본문으로 GoogleVerify를 호출합니다
func callGoogleVerifyWithBody(t *testing.T, handler *AuthHandler, requestBody map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/google/verify", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.GoogleVerify(rec, req)
	return rec
}

// decodeGoogleVerifyResponse는 성공 응답 본문을 해석합니다
func decodeGoogleVerifyResponse(t *testing.T, rec *httptest.ResponseRecorder) GoogleVerifyResponse {
	t.Helper()
	var body GoogleVerifyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v, body: %s", err, rec.Body.String())
	}
	return body
}

// countUsersByGoogleID는 google_id 사용자의 users 행 수를 셉니다
func countUsersByGoogleID(t *testing.T, db *sql.DB, googleID string) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM users WHERE google_id = $1`, googleID).Scan(&count); err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	return count
}

// countRefreshTokensByGoogleID는 google_id 사용자의 refresh_tokens 행 수를 셉니다
func countRefreshTokensByGoogleID(t *testing.T, db *sql.DB, googleID string) int {
	t.Helper()
	var count int
	query := `SELECT COUNT(*) FROM refresh_tokens rt JOIN users u ON u.id = rt.user_id WHERE u.google_id = $1`
	if err := db.QueryRowContext(context.Background(), query, googleID).Scan(&count); err != nil {
		t.Fatalf("failed to count refresh tokens: %v", err)
	}
	return count
}

// TestGoogleVerify_NewUserSuccess는 신규 사용자 로그인 시 사용자 생성과 세션 발급 결과를 테스트합니다
func TestGoogleVerify_NewUserSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping committed login test in short mode")
	}
	db := testhelpers.SetupTestDB(t)
	t.Cleanup(func() { database.Close(db) })

	// Given: 아직 가입하지 않은 사용자의 검증된 Google ID Token
	googleID, email := newGoogleVerifyTestIdentity(t, db)
	handler := newGoogleVerifyTestHandler(db, googleID, email)

	// When: GoogleVerify 호출
	rec := callGoogleVerify(t, handler, email)

	// Then: 200과 토큰 쌍, 생성된 사용자, 캐시 금지 헤더
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store")
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	for _, field := range []string{"access_token_expires_at", "refresh_token_expires_at"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("response is missing %q", field)
		}
	}
	if _, ok := raw["token"]; ok {
		t.Error("response must not include the token field (use access_token)")
	}

	body := decodeGoogleVerifyResponse(t, rec)
	if body.TokenType != "Bearer" {
		t.Errorf("token_type = %q, want %q", body.TokenType, "Bearer")
	}
	if body.AccessToken == "" {
		t.Error("access_token is empty")
	}
	if body.RefreshToken == "" {
		t.Error("refresh_token is empty")
	}

	// Then: Refresh Token 만료는 주입한 시각 + 기본 유휴 수명(14일)
	wantRefreshExpiresAt := googleVerifyTestTime.Add(14 * 24 * time.Hour)
	if !body.RefreshTokenExpiresAt.Equal(wantRefreshExpiresAt) {
		t.Errorf("refresh_token_expires_at = %v, want %v", body.RefreshTokenExpiresAt, wantRefreshExpiresAt)
	}

	// Then: Access Token 만료는 실제 현재 시각 + 168시간 근처 (발급 함수가 실제 시각을 사용함)
	wantAccessExpiresAt := time.Now().Add(168 * time.Hour)
	if diff := body.AccessTokenExpiresAt.Sub(wantAccessExpiresAt).Abs(); diff > time.Minute {
		t.Errorf("access_token_expires_at = %v, want within 1m of %v", body.AccessTokenExpiresAt, wantAccessExpiresAt)
	}

	stored, err := database.GetUserByGoogleID(context.Background(), db, googleID)
	if err != nil {
		t.Fatalf("failed to get created user: %v", err)
	}
	if stored == nil {
		t.Fatal("user was not created")
	}
	if body.User == nil || body.User.ID != stored.ID {
		t.Errorf("response user = %+v, want id %d", body.User, stored.ID)
	}
	if count := countRefreshTokensByGoogleID(t, db, googleID); count != 1 {
		t.Errorf("refresh_tokens rows = %d, want 1", count)
	}
}

// TestGoogleVerify_ExistingUserStartsNewFamily는 같은 사용자가 다시 로그인하면 새 계열이 발급되는지 테스트합니다
func TestGoogleVerify_ExistingUserStartsNewFamily(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping committed login test in short mode")
	}
	db := testhelpers.SetupTestDB(t)
	t.Cleanup(func() { database.Close(db) })

	// Given: 한 번 로그인한 사용자
	googleID, email := newGoogleVerifyTestIdentity(t, db)
	handler := newGoogleVerifyTestHandler(db, googleID, email)
	first := callGoogleVerify(t, handler, email)
	if first.Code != http.StatusOK {
		t.Fatalf("first login: expected status %d, got %d. Body: %s", http.StatusOK, first.Code, first.Body.String())
	}
	firstBody := decodeGoogleVerifyResponse(t, first)

	// When: 같은 사용자가 다시 로그인
	second := callGoogleVerify(t, handler, email)

	// Then: 200, 새 Refresh Token, 서로 다른 계열 두 개
	if second.Code != http.StatusOK {
		t.Fatalf("second login: expected status %d, got %d. Body: %s", http.StatusOK, second.Code, second.Body.String())
	}
	secondBody := decodeGoogleVerifyResponse(t, second)
	if secondBody.RefreshToken == firstBody.RefreshToken {
		t.Error("second login returned the same refresh_token")
	}
	if secondBody.User == nil || firstBody.User == nil || secondBody.User.ID != firstBody.User.ID {
		t.Errorf("second login user = %+v, want same user as first %+v", secondBody.User, firstBody.User)
	}

	var familyCount int
	query := `SELECT COUNT(DISTINCT rt.family_id) FROM refresh_tokens rt JOIN users u ON u.id = rt.user_id WHERE u.google_id = $1`
	if err := db.QueryRowContext(context.Background(), query, googleID).Scan(&familyCount); err != nil {
		t.Fatalf("failed to count families: %v", err)
	}
	if familyCount != 2 {
		t.Errorf("distinct family_id = %d, want 2", familyCount)
	}
}

// TestGoogleVerify_SessionFailureRollsBackUser는 세션 발급이 실패하면 새 사용자 생성까지 되돌리는지 테스트합니다
func TestGoogleVerify_SessionFailureRollsBackUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping committed login test in short mode")
	}
	db := testhelpers.SetupTestDB(t)
	t.Cleanup(func() { database.Close(db) })

	// Given: 신규 사용자, Access Token 발급이 실패하는 짧은 JWT_SECRET
	googleID, email := newGoogleVerifyTestIdentity(t, db)
	handler := newGoogleVerifyTestHandler(db, googleID, email)
	t.Setenv("JWT_SECRET", "short")

	// When: GoogleVerify 호출
	rec := callGoogleVerify(t, handler, email)

	// Then: 500이고, 사용자와 Refresh Token이 모두 남지 않음
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
	if code := readErrorCode(t, rec); code != ErrInternalServer {
		t.Errorf("error.code = %q, want %q", code, ErrInternalServer)
	}
	if count := countUsersByGoogleID(t, db, googleID); count != 0 {
		t.Errorf("users rows = %d, want 0", count)
	}
	if count := countRefreshTokensByGoogleID(t, db, googleID); count != 0 {
		t.Errorf("refresh_tokens rows = %d, want 0", count)
	}
}

// mustGetUserByGoogleID는 google_id 사용자를 조회하고, 없으면 테스트를 중단합니다
func mustGetUserByGoogleID(t *testing.T, db *sql.DB, googleID string) *models.User {
	t.Helper()
	user, err := database.GetUserByGoogleID(context.Background(), db, googleID)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if user == nil {
		t.Fatalf("user %s was not created", googleID)
	}
	return user
}

// TestGoogleVerify_UsesVerifiedClaims는 사용자 생성 시 요청 본문보다 ID Token의 검증된 값을 우선 쓰는지 테스트합니다
func TestGoogleVerify_UsesVerifiedClaims(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping committed login test in short mode")
	}
	db := testhelpers.SetupTestDB(t)
	t.Cleanup(func() { database.Close(db) })

	t.Run("요청 본문 email이 ID Token과 달라도 ID Token의 이메일·이름·사진으로 저장한다", func(t *testing.T) {
		// Given: 검증된 payload와 다른 값을 담은 요청 본문
		googleID, email := newGoogleVerifyTestIdentity(t, db)
		handler := newGoogleVerifyTestHandlerWithPayload(db, auth.GoogleIDTokenPayload{
			GoogleID: googleID,
			Email:    email,
			Name:     "Verified Name",
			Picture:  "https://example.com/verified.jpg",
		})

		// When: 다른 이메일·이름·사진으로 로그인 요청
		rec := callGoogleVerifyWithBody(t, handler, map[string]interface{}{
			"id_token": "fake-google-id-token",
			"email":    "attacker-" + email,
			"name":     "Request Name",
			"picture":  "https://example.com/request.jpg",
		})

		// Then: 200이고 저장된 사용자와 응답은 payload 값
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		stored := mustGetUserByGoogleID(t, db, googleID)
		if stored.Email != email || stored.Name != "Verified Name" || stored.PictureURL != "https://example.com/verified.jpg" {
			t.Errorf("stored user = %+v, want payload values", stored)
		}
		body := decodeGoogleVerifyResponse(t, rec)
		if body.User == nil || body.User.Email != email {
			t.Errorf("response user = %+v, want email %q", body.User, email)
		}
	})

	t.Run("id_token만 보내도 로그인에 성공한다", func(t *testing.T) {
		// Given: 이메일·이름이 있는 payload, id_token만 담은 요청
		googleID, email := newGoogleVerifyTestIdentity(t, db)
		handler := newGoogleVerifyTestHandlerWithPayload(db, auth.GoogleIDTokenPayload{GoogleID: googleID, Email: email, Name: "Verified Name"})

		// When: 로그인 요청
		rec := callGoogleVerifyWithBody(t, handler, map[string]interface{}{"id_token": "fake-google-id-token"})

		// Then: 200이고 payload 값으로 저장
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		stored := mustGetUserByGoogleID(t, db, googleID)
		if stored.Email != email || stored.Name != "Verified Name" {
			t.Errorf("stored user = %+v", stored)
		}
	})

	t.Run("ID Token에 이름·사진이 없으면 요청 본문 값을 쓴다", func(t *testing.T) {
		// Given: 이름·사진이 없는 payload
		googleID, email := newGoogleVerifyTestIdentity(t, db)
		handler := newGoogleVerifyTestHandlerWithPayload(db, auth.GoogleIDTokenPayload{GoogleID: googleID, Email: email})

		// When: 이름·사진을 담은 요청
		rec := callGoogleVerifyWithBody(t, handler, map[string]interface{}{
			"id_token": "fake-google-id-token",
			"name":     "Request Name",
			"picture":  "https://example.com/request.jpg",
		})

		// Then: 요청 본문의 이름·사진으로 저장
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		stored := mustGetUserByGoogleID(t, db, googleID)
		if stored.Name != "Request Name" || stored.PictureURL != "https://example.com/request.jpg" {
			t.Errorf("stored user = %+v, want request name and picture", stored)
		}
	})

	t.Run("ID Token과 요청 본문 모두 이름이 없으면 이메일을 이름으로 쓴다", func(t *testing.T) {
		// Given: 이름이 없는 payload
		googleID, email := newGoogleVerifyTestIdentity(t, db)
		handler := newGoogleVerifyTestHandlerWithPayload(db, auth.GoogleIDTokenPayload{GoogleID: googleID, Email: email})

		// When: 이름 없는 요청
		rec := callGoogleVerifyWithBody(t, handler, map[string]interface{}{"id_token": "fake-google-id-token"})

		// Then: 이메일이 이름으로 저장됨
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		stored := mustGetUserByGoogleID(t, db, googleID)
		if stored.Name != email {
			t.Errorf("Name = %q, want %q", stored.Name, email)
		}
	})

	t.Run("100자를 넘는 이름은 룬 기준 100자로 잘라 저장한다", func(t *testing.T) {
		// Given: 멀티바이트 문자 150자로 된 이름의 payload
		googleID, email := newGoogleVerifyTestIdentity(t, db)
		longName := strings.Repeat("가", 150)
		handler := newGoogleVerifyTestHandlerWithPayload(db, auth.GoogleIDTokenPayload{GoogleID: googleID, Email: email, Name: longName})

		// When: 로그인
		rec := callGoogleVerifyWithBody(t, handler, map[string]interface{}{"id_token": "fake-google-id-token"})

		// Then: 이름이 앞 100자로 잘려 저장됨
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		stored := mustGetUserByGoogleID(t, db, googleID)
		if want := strings.Repeat("가", maxUserNameLength); stored.Name != want {
			t.Errorf("Name = %q (%d runes), want %d runes of the original", stored.Name, utf8.RuneCountInString(stored.Name), maxUserNameLength)
		}
	})

	t.Run("이름 대신 쓰는 이메일이 100자를 넘으면 룬 기준 100자로 잘라 저장한다", func(t *testing.T) {
		// Given: 이름이 없고 100자를 넘는 이메일의 payload
		googleID, email := newGoogleVerifyTestIdentity(t, db)
		longEmail := strings.Repeat("a", 120) + email
		handler := newGoogleVerifyTestHandlerWithPayload(db, auth.GoogleIDTokenPayload{GoogleID: googleID, Email: longEmail})

		// When: 이름 없는 요청으로 로그인
		rec := callGoogleVerifyWithBody(t, handler, map[string]interface{}{"id_token": "fake-google-id-token"})

		// Then: 이메일은 그대로, 이름은 이메일의 앞 100자로 저장됨
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
		stored := mustGetUserByGoogleID(t, db, googleID)
		if stored.Email != longEmail {
			t.Errorf("Email = %q, want %q", stored.Email, longEmail)
		}
		if want := longEmail[:maxUserNameLength]; stored.Name != want {
			t.Errorf("Name = %q, want %q", stored.Name, want)
		}
	})
}

// TestGoogleVerify_EmailTakenByOtherGoogleAccount는 다른 Google 계정이 이미 쓰는 이메일로 첫 로그인하면 500을 반환하는지 테스트합니다
func TestGoogleVerify_EmailTakenByOtherGoogleAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping committed login test in short mode")
	}
	db := testhelpers.SetupTestDB(t)
	t.Cleanup(func() { database.Close(db) })

	// Given: 같은 이메일을 쓰는 다른 Google 계정의 사용자
	ownerGoogleID, email := newGoogleVerifyTestIdentity(t, db)
	owner := &models.User{Email: email, Name: "Owner", GoogleID: ownerGoogleID}
	if err := database.CreateUser(context.Background(), db, owner); err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}
	otherGoogleID, _ := newGoogleVerifyTestIdentity(t, db)
	handler := newGoogleVerifyTestHandler(db, otherGoogleID, email)

	// Given: 서버 로그를 가로채 확인
	var logs bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(originalOutput) })

	// When: 다른 Google 계정으로 첫 로그인
	rec := callGoogleVerify(t, handler, email)

	// Then: 500이고 새 사용자는 생기지 않음
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
	if code := readErrorCode(t, rec); code != ErrInternalServer {
		t.Errorf("error.code = %q, want %q", code, ErrInternalServer)
	}
	if count := countUsersByGoogleID(t, db, otherGoogleID); count != 0 {
		t.Errorf("users rows = %d, want 0", count)
	}

	// Then: 응답에는 이메일 점유 사실이 드러나지 않음
	if strings.Contains(rec.Body.String(), "email") {
		t.Errorf("response must not reveal email usage, got: %s", rec.Body.String())
	}

	// Then: 로그에는 Google ID와 이메일 점유 원인이 남음
	logged := logs.String()
	if !strings.Contains(logged, "google_id="+otherGoogleID) || !strings.Contains(logged, database.ErrEmailTaken.Error()) {
		t.Errorf("log = %q, want google_id and email-taken cause", logged)
	}
}
