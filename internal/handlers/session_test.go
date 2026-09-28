package handlers

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/june20516/orbithall/internal/auth"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/ratelimit"
	"github.com/june20516/orbithall/internal/testhelpers"
	"golang.org/x/time/rate"
)

func init() {
	// 테스트용 Refresh Token 비밀키 (JWT_SECRET과 다른 값)
	os.Setenv("REFRESH_TOKEN_SECRET", "test-refresh-secret-for-handlers-at-least-32-chars")
}

// sessionTestTime은 세션 테스트의 기준 시각입니다
var sessionTestTime = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// sessionTestConfig는 명세 기본값과 같은 수명 설정입니다
func sessionTestConfig() auth.RefreshTokenConfig {
	return auth.RefreshTokenConfig{
		IdleTTL:     14 * 24 * time.Hour,
		AbsoluteTTL: 30 * 24 * time.Hour,
		ReuseGrace:  30 * time.Second,
	}
}

// unlimitedRefreshLimiter는 요청 제한이 없는 limiter입니다
func unlimitedRefreshLimiter() *ratelimit.RateLimiter {
	return ratelimit.NewRateLimiter(rate.Inf, 1)
}

// createSessionTestUser는 세션 테스트용 사용자를 만듭니다
func createSessionTestUser(ctx context.Context, t *testing.T, tx database.DBTX) *models.User {
	t.Helper()
	user := &models.User{
		Email:    "session@example.com",
		Name:     "Session User",
		GoogleID: "google-session-user",
	}
	if err := database.CreateUser(ctx, tx, user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// mustIssueSession은 기준 시각에 로그인 세션을 발급합니다
func mustIssueSession(ctx context.Context, t *testing.T, tx database.DBTX, user *models.User, cfg auth.RefreshTokenConfig) *TokenPairResponse {
	t.Helper()
	pair, err := issueSession(ctx, tx, user, cfg, sessionTestTime)
	if err != nil {
		t.Fatalf("failed to issue session: %v", err)
	}
	return pair
}

// mustRotate는 회전이 성공해야 하는 경우에 사용합니다
func mustRotate(ctx context.Context, t *testing.T, tx database.DBTX, refreshToken string, cfg auth.RefreshTokenConfig, now time.Time) *TokenPairResponse {
	t.Helper()
	pair, err := rotateSession(ctx, tx, refreshToken, cfg, unlimitedRefreshLimiter(), now)
	if err != nil {
		t.Fatalf("expected rotation to succeed, got: %v", err)
	}
	return pair
}

// mustGetSessionToken은 저장된 Refresh Token을 원문으로 조회합니다
func mustGetSessionToken(ctx context.Context, t *testing.T, tx database.DBTX, refreshToken string) *models.RefreshToken {
	t.Helper()
	token, err := database.GetRefreshTokenByHash(ctx, tx, auth.HashRefreshToken(refreshToken))
	if err != nil || token == nil {
		t.Fatalf("failed to load refresh token: %+v, %v", token, err)
	}
	return token
}

// assertSessionFamilyNotRevoked는 refreshToken이 속한 계열이 폐기되지 않았는지 확인합니다
func assertSessionFamilyNotRevoked(ctx context.Context, t *testing.T, tx database.DBTX, refreshToken string) {
	t.Helper()
	token := mustGetSessionToken(ctx, t, tx, refreshToken)
	revoked, err := database.IsRefreshTokenFamilyRevoked(ctx, tx, token.FamilyID)
	if err != nil {
		t.Fatalf("failed to check family revocation: %v", err)
	}
	if revoked {
		t.Error("expected family not to be revoked")
	}
}

// TestIssueSession은 로그인 시 세션 발급을 테스트합니다
func TestIssueSession(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("토큰 쌍을 발급하고 Refresh Token 해시를 저장한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자
		user := createSessionTestUser(ctx, t, tx)

		// When: 세션 발급
		pair, err := issueSession(ctx, tx, user, sessionTestConfig(), sessionTestTime)

		// Then: 토큰 쌍 필드가 채워짐
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if pair.TokenType != "Bearer" || pair.AccessToken == "" || pair.AccessTokenExpiresAt.IsZero() {
			t.Errorf("unexpected access token fields: %+v", pair)
		}
		if !strings.HasPrefix(pair.RefreshToken, auth.RefreshTokenPrefix) {
			t.Errorf("refresh token has no prefix: %q", pair.RefreshToken)
		}
		wantRefreshExpiresAt := sessionTestTime.Add(14 * 24 * time.Hour)
		if !pair.RefreshTokenExpiresAt.Equal(wantRefreshExpiresAt) {
			t.Errorf("RefreshTokenExpiresAt = %v, want %v", pair.RefreshTokenExpiresAt, wantRefreshExpiresAt)
		}

		// Then: 원문이 아닌 해시로 저장되고 절대 만료는 30일 뒤
		stored, err := database.GetRefreshTokenByHash(ctx, tx, auth.HashRefreshToken(pair.RefreshToken))
		if err != nil || stored == nil {
			t.Fatalf("expected stored token, got %+v, %v", stored, err)
		}
		if stored.UserID != user.ID {
			t.Errorf("UserID = %d, want %d", stored.UserID, user.ID)
		}
		if !stored.FamilyExpiresAt.Equal(sessionTestTime.Add(30 * 24 * time.Hour)) {
			t.Errorf("FamilyExpiresAt = %v", stored.FamilyExpiresAt)
		}
	})

	t.Run("오래된 토큰을 정리한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 8일 전에 절대 만료된 계열
		user := createSessionTestUser(ctx, t, tx)
		staleExpiresAt := sessionTestTime.Add(-8 * 24 * time.Hour)
		if _, err := database.CreateRefreshTokenFamily(ctx, tx, user.ID, []byte("stale-hash"), staleExpiresAt, staleExpiresAt); err != nil {
			t.Fatalf("failed to create stale token: %v", err)
		}

		// When: 새 로그인
		mustIssueSession(ctx, t, tx, user, sessionTestConfig())

		// Then: 오래된 계열이 삭제됨
		stale, _ := database.GetRefreshTokenByHash(ctx, tx, []byte("stale-hash"))
		if stale != nil {
			t.Error("expected stale token to be deleted")
		}
	})
}

// TestRotateSession은 Refresh Token 회전 규칙을 테스트합니다
func TestRotateSession(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	cfg := sessionTestConfig()

	t.Run("새 토큰 쌍을 발급하고 이전 토큰을 사용 처리한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)

		// When: 1분 뒤 회전
		now := sessionTestTime.Add(time.Minute)
		pair, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), now)

		// Then: 새 Refresh Token과 연장된 만료
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if pair.RefreshToken == login.RefreshToken {
			t.Error("expected a new refresh token")
		}
		if pair.AccessToken == "" {
			t.Error("expected an access token")
		}
		if !pair.RefreshTokenExpiresAt.Equal(now.Add(cfg.IdleTTL)) {
			t.Errorf("RefreshTokenExpiresAt = %v, want %v", pair.RefreshTokenExpiresAt, now.Add(cfg.IdleTTL))
		}

		// Then: 이전 토큰은 사용 처리됨
		old, _ := database.GetRefreshTokenByHash(ctx, tx, auth.HashRefreshToken(login.RefreshToken))
		if old.UsedAt == nil {
			t.Error("expected previous token to be marked used")
		}
	})

	t.Run("회전한 토큰으로 다시 회전할 수 있다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 한 번 회전한 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, sessionTestTime.Add(time.Hour))

		// When: 하루 뒤 두 번째 토큰으로 회전
		third, err := rotateSession(ctx, tx, second.RefreshToken, cfg, unlimitedRefreshLimiter(), sessionTestTime.Add(25*time.Hour))

		// Then: 성공하고 또 다른 토큰
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if third.RefreshToken == second.RefreshToken || third.RefreshToken == login.RefreshToken {
			t.Error("expected a new refresh token")
		}
	})

	t.Run("유예 시간 안에 이전 토큰을 다시 내면 같은 후속 토큰을 돌려준다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		first := mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)

		// When: 10초 뒤 R1을 다시 제출
		again, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(10*time.Second))

		// Then: 같은 R2와 같은 만료 시각
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if again.RefreshToken != first.RefreshToken {
			t.Errorf("expected the same R2, got a different token")
		}
		if !again.RefreshTokenExpiresAt.Equal(first.RefreshTokenExpiresAt) {
			t.Errorf("RefreshTokenExpiresAt = %v, want %v", again.RefreshTokenExpiresAt, first.RefreshTokenExpiresAt)
		}
		if again.AccessToken == "" {
			t.Error("expected an access token")
		}
	})

	t.Run("유예 시간이 지나 이전 토큰을 다시 내면 계열 전체를 폐기한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)

		// When: 31초 뒤 R1을 다시 제출
		_, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(31*time.Second))

		// Then: 재사용 에러
		if !errors.Is(err, errSessionReused) {
			t.Fatalf("expected errSessionReused, got: %v", err)
		}

		// Then: R2도 폐기되어 쓸 수 없음
		_, err = rotateSession(ctx, tx, second.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(time.Minute))
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid for R2, got: %v", err)
		}
	})

	t.Run("후속 토큰이 이미 사용되었으면 유예 시간 안이라도 재사용으로 본다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 -> R3 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)
		mustRotate(ctx, t, tx, second.RefreshToken, cfg, rotatedAt.Add(5*time.Second))

		// When: R1을 10초 뒤 다시 제출
		_, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(10*time.Second))

		// Then: 재사용 에러
		if !errors.Is(err, errSessionReused) {
			t.Errorf("expected errSessionReused, got: %v", err)
		}
	})

	t.Run("계열이 폐기되었으면 유예 시간 안이라도 무효", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 -> R3 회전 후 계열 폐기
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)
		third := mustRotate(ctx, t, tx, second.RefreshToken, cfg, rotatedAt.Add(time.Second))
		first, err := database.GetRefreshTokenByHash(ctx, tx, auth.HashRefreshToken(login.RefreshToken))
		if err != nil || first == nil {
			t.Fatalf("failed to load first token: %+v, %v", first, err)
		}
		if err := database.RevokeRefreshTokenFamily(ctx, tx, first.FamilyID, models.RefreshTokenRevokedByReuse, rotatedAt.Add(2*time.Second)); err != nil {
			t.Fatalf("failed to revoke family: %v", err)
		}

		// Given: 동시 회전으로 폐기를 빠져나간 상황 재현 (R2, R3의 폐기 표시만 지움)
		for _, token := range []string{second.RefreshToken, third.RefreshToken} {
			if _, err := tx.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at = NULL, revoked_reason = NULL WHERE token_hash = $1`, auth.HashRefreshToken(token)); err != nil {
				t.Fatalf("failed to clear revocation: %v", err)
			}
		}

		// When: 유예 시간 안에 사용된 R2를 다시 제출 (후속 R3는 미사용)
		_, err = rotateSession(ctx, tx, second.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(5*time.Second))

		// Then: R3를 돌려주지 않고 무효 에러
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid, got: %v", err)
		}
	})

	t.Run("유휴 만료 시각이 되면 만료 에러", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)

		// When: 14일 뒤 회전
		_, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), sessionTestTime.Add(cfg.IdleTTL))

		// Then: 만료 에러
		if !errors.Is(err, errSessionExpired) {
			t.Errorf("expected errSessionExpired, got: %v", err)
		}
	})

	t.Run("절대 만료를 넘겨 연장되지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 유휴 30분, 절대 1시간 설정으로 로그인
		shortCfg := auth.RefreshTokenConfig{IdleTTL: 30 * time.Minute, AbsoluteTTL: time.Hour, ReuseGrace: 30 * time.Second}
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, shortCfg)

		// When: 20분, 45분에 회전
		second := mustRotate(ctx, t, tx, login.RefreshToken, shortCfg, sessionTestTime.Add(20*time.Minute))
		third := mustRotate(ctx, t, tx, second.RefreshToken, shortCfg, sessionTestTime.Add(45*time.Minute))

		// Then: 세 번째 토큰의 만료는 절대 만료(로그인 + 1시간)로 제한됨
		absoluteExpiresAt := sessionTestTime.Add(time.Hour)
		if !third.RefreshTokenExpiresAt.Equal(absoluteExpiresAt) {
			t.Errorf("RefreshTokenExpiresAt = %v, want %v", third.RefreshTokenExpiresAt, absoluteExpiresAt)
		}

		// Then: 절대 만료 시각에는 만료 에러
		_, err := rotateSession(ctx, tx, third.RefreshToken, shortCfg, unlimitedRefreshLimiter(), absoluteExpiresAt)
		if !errors.Is(err, errSessionExpired) {
			t.Errorf("expected errSessionExpired, got: %v", err)
		}
	})

	t.Run("모르는 토큰은 무효", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 저장되지 않은 토큰으로 회전
		_, err := rotateSession(ctx, tx, "ohrt_unknown", cfg, unlimitedRefreshLimiter(), sessionTestTime)

		// Then: 무효 에러
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid, got: %v", err)
		}
	})

	t.Run("계열당 요청이 너무 많으면 제한한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 한 시간에 한 번만 허용하는 limiter와 로그인 세션
		limiter := ratelimit.NewRateLimiter(rate.Every(time.Hour), 1)
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		second, err := rotateSession(ctx, tx, login.RefreshToken, cfg, limiter, sessionTestTime.Add(time.Minute))
		if err != nil {
			t.Fatalf("first rotation failed: %v", err)
		}

		// When: 같은 계열로 바로 다시 회전
		_, err = rotateSession(ctx, tx, second.RefreshToken, cfg, limiter, sessionTestTime.Add(2*time.Minute))

		// Then: 제한 에러
		if !errors.Is(err, errSessionRateLimited) {
			t.Errorf("expected errSessionRateLimited, got: %v", err)
		}
	})
	t.Run("폐기를 빠져나간 미사용 토큰으로는 회전되지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 -> R3 회전 후 계열 폐기
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)
		third := mustRotate(ctx, t, tx, second.RefreshToken, cfg, rotatedAt.Add(time.Second))
		first := mustGetSessionToken(ctx, t, tx, login.RefreshToken)
		if err := database.RevokeRefreshTokenFamily(ctx, tx, first.FamilyID, models.RefreshTokenRevokedByReuse, rotatedAt.Add(2*time.Second)); err != nil {
			t.Fatalf("failed to revoke family: %v", err)
		}

		// Given: 동시 회전으로 폐기를 빠져나간 상황 재현 (미사용 R3의 폐기 표시만 지움)
		if _, err := tx.ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at = NULL, revoked_reason = NULL WHERE token_hash = $1`, auth.HashRefreshToken(third.RefreshToken)); err != nil {
			t.Fatalf("failed to clear revocation: %v", err)
		}

		// When: R3로 회전 (DB 회전이 거부되어 재조회 경로를 탐)
		_, err := rotateSession(ctx, tx, third.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(5*time.Second))

		// Then: 무효 에러이고 R3의 후속 토큰은 없음
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid, got: %v", err)
		}
		reloadedThird := mustGetSessionToken(ctx, t, tx, third.RefreshToken)
		child, err := database.GetChildRefreshToken(ctx, tx, reloadedThird.ID)
		if err != nil || child != nil {
			t.Errorf("expected no child of R3, got %+v, %v", child, err)
		}
	})

	t.Run("사용 후 정확히 유예 시간이면 후속 토큰을 돌려주고 새 행을 만들지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)

		// When: 정확히 30초 뒤 R1을 다시 제출
		again, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(cfg.ReuseGrace))

		// Then: 같은 R2
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if again.RefreshToken != second.RefreshToken {
			t.Error("expected the same R2, got a different token")
		}

		// Then: R2의 후속 토큰이 생기지 않아 계열이 한 줄로 유지됨
		storedSecond := mustGetSessionToken(ctx, t, tx, second.RefreshToken)
		child, err := database.GetChildRefreshToken(ctx, tx, storedSecond.ID)
		if err != nil || child != nil {
			t.Errorf("expected no child of R2, got %+v, %v", child, err)
		}
		if storedSecond.UsedAt != nil {
			t.Error("expected R2 to remain unused")
		}
	})

	t.Run("사용 후 유예 시간을 1ns 넘기면 재사용으로 본다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)

		// When: 30초 + 1ns 뒤 R1을 다시 제출
		_, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(cfg.ReuseGrace+time.Nanosecond))

		// Then: 재사용 에러
		if !errors.Is(err, errSessionReused) {
			t.Errorf("expected errSessionReused, got: %v", err)
		}
	})

	t.Run("유예 시간 안이라도 후속 토큰이 만료되었으면 만료 에러이고 계열을 폐기하지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 절대 만료 1시간 설정에서 만료 5초 전에 R1 -> R2 회전 (R1, R2 만료 = 절대 만료)
		shortCfg := auth.RefreshTokenConfig{IdleTTL: 2 * time.Hour, AbsoluteTTL: time.Hour, ReuseGrace: 30 * time.Second}
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, shortCfg)
		absoluteExpiresAt := sessionTestTime.Add(time.Hour)
		mustRotate(ctx, t, tx, login.RefreshToken, shortCfg, absoluteExpiresAt.Add(-5*time.Second))

		// When: 절대 만료 1초 뒤(사용 6초 뒤) R1을 다시 제출
		_, err := rotateSession(ctx, tx, login.RefreshToken, shortCfg, unlimitedRefreshLimiter(), absoluteExpiresAt.Add(time.Second))

		// Then: 만료 에러이고 계열은 폐기되지 않음
		if !errors.Is(err, errSessionExpired) {
			t.Errorf("expected errSessionExpired, got: %v", err)
		}
		assertSessionFamilyNotRevoked(ctx, t, tx, login.RefreshToken)
	})

	t.Run("비밀키가 바뀌어 후속 토큰을 다시 계산할 수 없으면 무효이고 계열을 폐기하지 않는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		rotatedAt := sessionTestTime.Add(time.Minute)
		mustRotate(ctx, t, tx, login.RefreshToken, cfg, rotatedAt)

		// Given: 유예 시간 안에 비밀키 교체
		t.Setenv("REFRESH_TOKEN_SECRET", "rotated-refresh-secret-for-handlers-at-least-32-chars")

		// When: 10초 뒤 R1을 다시 제출
		_, err := rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), rotatedAt.Add(10*time.Second))

		// Then: 무효 에러이고 계열은 폐기되지 않음
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid, got: %v", err)
		}
		assertSessionFamilyNotRevoked(ctx, t, tx, login.RefreshToken)
	})
}

// TestRevokeSession은 로그아웃을 테스트합니다
func TestRevokeSession(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	cfg := sessionTestConfig()

	t.Run("계열을 폐기해 이후 회전을 막는다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그인 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)

		// When: 로그아웃
		err := revokeSession(ctx, tx, login.RefreshToken, sessionTestTime.Add(time.Minute))

		// Then: 에러 없고 이후 회전은 무효
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		_, err = rotateSession(ctx, tx, login.RefreshToken, cfg, unlimitedRefreshLimiter(), sessionTestTime.Add(2*time.Minute))
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid, got: %v", err)
		}
	})

	t.Run("모르는 토큰이어도 에러 없이 끝난다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// When: 저장되지 않은 토큰으로 로그아웃
		err := revokeSession(ctx, tx, "ohrt_unknown", sessionTestTime)

		// Then: 에러 없음
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("사용된 이전 토큰으로 로그아웃해도 후속 토큰까지 폐기한다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: R1 -> R2 회전
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		second := mustRotate(ctx, t, tx, login.RefreshToken, cfg, sessionTestTime.Add(time.Minute))

		// When: 사용된 R1으로 로그아웃
		err := revokeSession(ctx, tx, login.RefreshToken, sessionTestTime.Add(2*time.Minute))

		// Then: 에러 없고 R2로도 회전할 수 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		_, err = rotateSession(ctx, tx, second.RefreshToken, cfg, unlimitedRefreshLimiter(), sessionTestTime.Add(3*time.Minute))
		if !errors.Is(err, errSessionInvalid) {
			t.Errorf("expected errSessionInvalid for R2, got: %v", err)
		}
	})

	t.Run("이미 폐기된 토큰으로 다시 로그아웃해도 에러 없이 끝난다", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 로그아웃한 세션
		user := createSessionTestUser(ctx, t, tx)
		login := mustIssueSession(ctx, t, tx, user, cfg)
		if err := revokeSession(ctx, tx, login.RefreshToken, sessionTestTime.Add(time.Minute)); err != nil {
			t.Fatalf("failed to revoke session: %v", err)
		}

		// When: 같은 토큰으로 다시 로그아웃
		err := revokeSession(ctx, tx, login.RefreshToken, sessionTestTime.Add(2*time.Minute))

		// Then: 에러 없음
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}
