package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

// sessionLockWaitTimeout은 다른 세션이 잠금을 기다리기 시작할 때까지 기다리는 최대 시간입니다
const sessionLockWaitTimeout = 5 * time.Second

// sessionLockWaitPollInterval은 잠금 대기 여부를 확인하는 간격입니다
const sessionLockWaitPollInterval = 10 * time.Millisecond

// createCommittedSessionTestUser는 커밋되는 고유 테스트 사용자를 만들고, 테스트 종료 시 삭제합니다
// 사용자를 지우면 CASCADE로 그 사용자의 토큰도 함께 삭제됩니다
func createCommittedSessionTestUser(ctx context.Context, t *testing.T, db *sql.DB) *models.User {
	t.Helper()
	suffix := fmt.Sprintf("session-concurrency-%d", time.Now().UnixNano())
	user := &models.User{
		Email:    suffix + "@example.com",
		Name:     "Session Concurrency User",
		GoogleID: "google-" + suffix,
	}
	if err := database.CreateUser(ctx, db, user); err != nil {
		t.Fatalf("failed to create committed test user: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID); err != nil {
			t.Errorf("failed to delete committed test user: %v", err)
		}
	})
	return user
}

// openSessionDedicatedConn은 풀에서 연결 하나를 꺼내 고정하고, 그 연결의 백엔드 프로세스 ID를 함께 반환합니다
func openSessionDedicatedConn(ctx context.Context, t *testing.T, db *sql.DB) (*sql.Conn, int) {
	t.Helper()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("failed to open dedicated connection: %v", err)
	}
	var pid int
	if err := conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		conn.Close()
		t.Fatalf("failed to get backend pid: %v", err)
	}
	return conn, pid
}

// waitForSessionLockWait는 pid 세션이 refresh_tokens 쿼리를 실행하면서 잠금을 기다리는 상태가 될 때까지 폴링합니다
func waitForSessionLockWait(ctx context.Context, t *testing.T, db *sql.DB, pid int) {
	t.Helper()
	query := `
		SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity
			WHERE pid = $1
				AND datname = current_database()
				AND wait_event_type = 'Lock'
				AND query ILIKE '%refresh_tokens%'
		)
	`

	deadline := time.Now().Add(sessionLockWaitTimeout)
	for time.Now().Before(deadline) {
		var waiting bool
		if err := db.QueryRowContext(ctx, query, pid).Scan(&waiting); err != nil {
			t.Fatalf("failed to poll pg_stat_activity: %v", err)
		}
		if waiting {
			return
		}
		time.Sleep(sessionLockWaitPollInterval)
	}
	t.Fatalf("session %d did not start waiting for a lock within %v", pid, sessionLockWaitTimeout)
}

// sessionRotateResult는 goroutine에서 실행한 세션 회전의 결과입니다
type sessionRotateResult struct {
	pair *TokenPairResponse
	err  error
}

// TestRotateSession_ConcurrentSameToken은 같은 Refresh Token으로 두 요청이 동시에 갱신할 때
// 둘 다 성공하고 같은 후속 토큰을 받는지 테스트합니다 (명세 8장 엣지 1)
func TestRotateSession_ConcurrentSameToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping session concurrency test in short mode")
	}
	db := testhelpers.SetupTestDB(t)
	t.Cleanup(func() { database.Close(db) })
	ctx := context.Background()
	cfg := sessionTestConfig()

	// Given: 커밋된 로그인 세션 R1
	user := createCommittedSessionTestUser(ctx, t, db)
	login, err := issueSession(ctx, db, user, cfg, sessionTestTime)
	if err != nil {
		t.Fatalf("failed to issue session: %v", err)
	}
	now := sessionTestTime.Add(time.Minute)

	// Given: 트랜잭션 A가 R1을 회전하고 커밋하지 않은 채 R1의 행 잠금을 보유
	txA, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction A: %v", err)
	}
	pairA, err := rotateSession(ctx, txA, login.RefreshToken, cfg, unlimitedRefreshLimiter(), now)
	if err != nil {
		txA.Rollback()
		t.Fatalf("expected rotation in A to succeed, got: %v", err)
	}

	// When: 다른 연결 B에서 같은 R1으로 회전 (A의 잠금을 기다림)
	connB, pidB := openSessionDedicatedConn(ctx, t, db)
	defer connB.Close()
	resultB := make(chan sessionRotateResult, 1)
	doneB := make(chan struct{})
	go func() {
		defer close(doneB)
		pair, err := rotateSession(ctx, connB, login.RefreshToken, cfg, unlimitedRefreshLimiter(), now)
		resultB <- sessionRotateResult{pair: pair, err: err}
	}()
	// 중간에 테스트가 중단되어도 A를 끝내 B가 풀려나고, B가 끝난 뒤에 연결을 닫습니다
	defer func() {
		txA.Rollback()
		<-doneB
	}()

	waitForSessionLockWait(ctx, t, db, pidB)
	if err := txA.Commit(); err != nil {
		t.Fatalf("failed to commit transaction A: %v", err)
	}
	result := <-resultB

	// Then: B도 에러 없이 A와 같은 R2를 받음
	if result.err != nil {
		t.Fatalf("expected rotation in B to succeed, got: %v", result.err)
	}
	if result.pair.RefreshToken != pairA.RefreshToken {
		t.Error("expected B to receive the same R2 as A")
	}
	if result.pair.AccessToken == "" {
		t.Error("expected an access token for B")
	}

	// Then: R1의 후속 토큰은 하나뿐
	first := mustGetSessionToken(ctx, t, db, login.RefreshToken)
	var childCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM refresh_tokens WHERE parent_id = $1`, first.ID).Scan(&childCount); err != nil {
		t.Fatalf("failed to count child refresh tokens: %v", err)
	}
	if childCount != 1 {
		t.Errorf("child count = %d, want 1", childCount)
	}
}
