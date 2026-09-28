package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/ratelimit"
	"github.com/june20516/orbithall/internal/testhelpers"
	"golang.org/x/time/rate"
)

// newTestSessionHandler는 현재 시각을 now로 고정하고 명세 기본 수명 설정을 쓰는 SessionHandler를 만듭니다
// 환경변수에 따라 수명 설정이 달라지지 않도록 설정을 직접 주입합니다
func newTestSessionHandler(tx database.DBTX, now time.Time) *SessionHandler {
	handler := NewSessionHandler(tx)
	handler.refreshConfig = sessionTestConfig()
	handler.now = func() time.Time { return now }
	return handler
}

// postSessionRequest는 refresh_token을 담은 JSON 요청을 handler에 보냅니다
func postSessionRequest(handler http.HandlerFunc, path string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

// refreshTokenBody는 {"refresh_token": token} JSON 문자열을 만듭니다
func refreshTokenBody(t *testing.T, token string) string {
	t.Helper()
	body, err := json.Marshal(RefreshTokenRequest{RefreshToken: token})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	return string(body)
}

// TestSessionHandler_Refresh는 POST /auth/refresh를 테스트합니다
func TestSessionHandler_Refresh(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("성공하면 200과 새 토큰 쌍", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		handler := newTestSessionHandler(tx, sessionTestTime.Add(time.Minute))

		// When: 갱신 요청
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))

		// Then: 200과 토큰 쌍
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
		}
		var pair TokenPairResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &pair); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if pair.TokenType != "Bearer" || pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Errorf("unexpected response: %+v", pair)
		}
		if pair.RefreshToken == login.RefreshToken {
			t.Error("expected a new refresh token")
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q, want no-store", got)
		}
	})

	t.Run("refresh_token이 없으면 400 INVALID_INPUT", func(t *testing.T) {
		_, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 빈 본문과 잘못된 JSON
		handler := newTestSessionHandler(tx, sessionTestTime)
		for _, body := range []string{`{}`, `{invalid`} {
			rec := postSessionRequest(handler.Refresh, "/auth/refresh", body)

			// Then: 400
			if rec.Code != http.StatusBadRequest {
				t.Errorf("body %s: status = %d", body, rec.Code)
			}
			if code := readErrorCode(t, rec); code != ErrInvalidInput {
				t.Errorf("body %s: error.code = %q", body, code)
			}
		}
	})

	t.Run("본문이 4KB를 넘으면 400 INVALID_INPUT", func(t *testing.T) {
		_, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 4KB를 넘는 refresh_token을 담은 본문
		handler := newTestSessionHandler(tx, sessionTestTime)
		oversizedBody := refreshTokenBody(t, "ohrt_"+strings.Repeat("a", maxSessionRequestBytes))

		// When: 갱신 요청
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", oversizedBody)

		// Then: 400 INVALID_INPUT
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrInvalidInput {
			t.Errorf("error.code = %q", code)
		}
	})

	t.Run("유예 시간 안에 이전 토큰으로 다시 요청하면 200과 같은 후속 토큰", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 핸들러로 한 번 갱신한 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		rotatedAt := sessionTestTime.Add(time.Minute)
		handler := newTestSessionHandler(tx, rotatedAt)
		firstRec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))
		if firstRec.Code != http.StatusOK {
			t.Fatalf("first refresh status = %d, body: %s", firstRec.Code, firstRec.Body.String())
		}
		var firstPair TokenPairResponse
		if err := json.Unmarshal(firstRec.Body.Bytes(), &firstPair); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		// When: 10초 뒤 이전 토큰으로 다시 갱신
		handler.now = func() time.Time { return rotatedAt.Add(10 * time.Second) }
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))

		// Then: 200과 첫 갱신 응답과 같은 refresh_token
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
		}
		var pair TokenPairResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &pair); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if pair.RefreshToken != firstPair.RefreshToken {
			t.Error("expected the same refresh token as the first refresh response")
		}
	})

	t.Run("모르는 토큰은 401 INVALID_REFRESH_TOKEN", func(t *testing.T) {
		_, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 저장되지 않은 토큰으로 갱신
		handler := newTestSessionHandler(tx, sessionTestTime)
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, "ohrt_unknown"))

		// Then: 401
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrInvalidRefreshToken {
			t.Errorf("error.code = %q", code)
		}
	})

	t.Run("만료된 토큰은 401 REFRESH_TOKEN_EXPIRED", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션, 15일 뒤
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		handler := newTestSessionHandler(tx, sessionTestTime.Add(15*24*time.Hour))

		// When: 갱신
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))

		// Then: 401
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrRefreshTokenExpired {
			t.Errorf("error.code = %q", code)
		}
	})

	t.Run("유예 시간 뒤 재사용은 401 REFRESH_TOKEN_REUSED", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 한 번 회전한 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		mustRotate(ctx, t, tx, login.RefreshToken, sessionTestConfig(), sessionTestTime)

		// When: 1분 뒤 이전 토큰으로 갱신
		handler := newTestSessionHandler(tx, sessionTestTime.Add(time.Minute))
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))

		// Then: 401
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrRefreshTokenReused {
			t.Errorf("error.code = %q", code)
		}
	})

	t.Run("요청 제한을 넘으면 429와 Retry-After", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 요청을 모두 거부하는 limiter
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		handler := newTestSessionHandler(tx, sessionTestTime)
		handler.limiter = ratelimit.NewRateLimiter(rate.Every(time.Hour), 0)

		// When: 갱신
		rec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))

		// Then: 429
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrRateLimitExceeded {
			t.Errorf("error.code = %q", code)
		}
		if got := rec.Header().Get("Retry-After"); got != "6" {
			t.Errorf("Retry-After = %q, want %q", got, "6")
		}
	})
}

// TestSessionHandler_Logout은 POST /auth/logout을 테스트합니다
func TestSessionHandler_Logout(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("204를 반환하고 이후 갱신을 막는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, sessionTestConfig())
		handler := newTestSessionHandler(tx, sessionTestTime.Add(time.Minute))

		// When: 로그아웃
		rec := postSessionRequest(handler.Logout, "/auth/logout", refreshTokenBody(t, login.RefreshToken))

		// Then: 204, 본문 없음
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, body: %s", rec.Code, rec.Body.String())
		}
		if rec.Body.Len() != 0 {
			t.Errorf("expected empty body, got %s", rec.Body.String())
		}

		// Then: 같은 토큰으로 갱신하면 401 INVALID_REFRESH_TOKEN
		refreshRec := postSessionRequest(handler.Refresh, "/auth/refresh", refreshTokenBody(t, login.RefreshToken))
		if code := readErrorCode(t, refreshRec); code != ErrInvalidRefreshToken {
			t.Errorf("error.code after logout = %q", code)
		}
	})

	t.Run("모르는 토큰이어도 204", func(t *testing.T) {
		_, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 저장되지 않은 토큰으로 로그아웃
		handler := newTestSessionHandler(tx, sessionTestTime)
		rec := postSessionRequest(handler.Logout, "/auth/logout", refreshTokenBody(t, "ohrt_unknown"))

		// Then: 204
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d", rec.Code)
		}
	})

	t.Run("refresh_token이 없으면 400 INVALID_INPUT", func(t *testing.T) {
		_, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 빈 본문으로 로그아웃
		handler := newTestSessionHandler(tx, sessionTestTime)
		rec := postSessionRequest(handler.Logout, "/auth/logout", `{}`)

		// Then: 400
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d", rec.Code)
		}
		if code := readErrorCode(t, rec); code != ErrInvalidInput {
			t.Errorf("error.code = %q", code)
		}
	})
}
