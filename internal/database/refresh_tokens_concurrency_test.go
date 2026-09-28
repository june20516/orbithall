package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

// lockWaitTimeout은 다른 세션이 잠금을 기다리기 시작할 때까지 기다리는 최대 시간입니다
const lockWaitTimeout = 5 * time.Second

// lockWaitPollInterval은 잠금 대기 여부를 확인하는 간격입니다
const lockWaitPollInterval = 10 * time.Millisecond

// concurrencyTestTimeout은 동시성 테스트 한 건의 DB 작업 전체에 거는 시간 제한입니다
// 잠금 대기가 풀리지 않는 경우에도 테스트가 무한히 멈추지 않게 합니다
const concurrencyTestTimeout = 30 * time.Second

// setupRefreshTokenConcurrencyTest는 실제로 커밋되는 데이터를 쓰는 동시성 테스트의 DB, 시간 제한 context, 고유 식별자를 준비합니다
// 반환하는 suffix는 사용자와 토큰 해시를 다른 실행과 겹치지 않게 만드는 데 씁니다
// context 취소는 t.Cleanup으로 등록하므로, 이후 등록되는 정리 작업(트랜잭션 롤백 등)이 모두 끝난 뒤에 실행됩니다
func setupRefreshTokenConcurrencyTest(t *testing.T) (context.Context, *sql.DB, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping refresh token concurrency test in short mode")
	}

	db := testhelpers.SetupTestDB(t)
	t.Cleanup(func() { Close(db) })

	ctx, cancel := context.WithTimeout(context.Background(), concurrencyTestTimeout)
	t.Cleanup(cancel)

	return ctx, db, fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano())
}

// beginLockHoldingTx는 행 잠금을 쥘 트랜잭션을 시작하고, 테스트 종료 시 롤백하도록 등록합니다
// 사용자를 만든 뒤에 호출하면 t.Cleanup이 역순으로 실행되므로, 테스트가 어디서 중단되어도
// 롤백이 사용자 삭제보다 먼저 실행되어 삭제가 이 트랜잭션의 잠금에 막히지 않습니다
func beginLockHoldingTx(ctx context.Context, t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin transaction A: %v", err)
	}
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

// createCommittedRefreshTokenTestUser는 커밋되는 테스트 사용자를 만들고, 테스트 종료 시 삭제합니다
// 사용자를 지우면 CASCADE로 그 사용자의 토큰도 함께 삭제됩니다
func createCommittedRefreshTokenTestUser(ctx context.Context, t *testing.T, db *sql.DB, suffix string) *models.User {
	t.Helper()
	user := &models.User{
		Email:    suffix + "@example.com",
		Name:     "Refresh Token Concurrency User",
		GoogleID: "google-" + suffix,
	}
	if err := CreateUser(ctx, db, user); err != nil {
		t.Fatalf("failed to create committed test user: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID); err != nil {
			t.Errorf("failed to delete committed test user: %v", err)
		}
	})
	return user
}

// openDedicatedConn은 풀에서 연결 하나를 꺼내 고정하고, 그 연결의 백엔드 프로세스 ID를 함께 반환합니다
func openDedicatedConn(ctx context.Context, t *testing.T, db *sql.DB) (*sql.Conn, int) {
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

// waitForLockWait는 pid 세션이 refresh_tokens 쿼리를 실행하면서 잠금을 기다리는 상태가 될 때까지 폴링합니다
// lockWaitTimeout 안에 대기 상태가 되지 않으면 테스트를 중단합니다
func waitForLockWait(ctx context.Context, t *testing.T, db *sql.DB, pid int) {
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

	deadline := time.Now().Add(lockWaitTimeout)
	for time.Now().Before(deadline) {
		var waiting bool
		if err := db.QueryRowContext(ctx, query, pid).Scan(&waiting); err != nil {
			t.Fatalf("failed to poll pg_stat_activity: %v", err)
		}
		if waiting {
			return
		}
		time.Sleep(lockWaitPollInterval)
	}
	t.Fatalf("session %d did not start waiting for a lock within %v", pid, lockWaitTimeout)
}

// countChildRefreshTokens는 parentID 토큰의 후속 토큰 수를 셉니다
func countChildRefreshTokens(ctx context.Context, t *testing.T, db *sql.DB, parentID int64) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM refresh_tokens WHERE parent_id = $1`, parentID).Scan(&count); err != nil {
		t.Fatalf("failed to count child refresh tokens: %v", err)
	}
	return count
}

// rotateResult는 goroutine에서 실행한 회전의 결과입니다
type rotateResult struct {
	token *models.RefreshToken
	err   error
}

// TestRotateRefreshToken_ConcurrentDoubleRotation은 서로 다른 연결에서 같은 토큰을 동시에 회전할 때
// 한 요청만 성공하는지 테스트합니다
func TestRotateRefreshToken_ConcurrentDoubleRotation(t *testing.T) {
	ctx, db, suffix := setupRefreshTokenConcurrencyTest(t)

	// Given: 커밋된 첫 토큰 P
	user := createCommittedRefreshTokenTestUser(ctx, t, db, suffix)
	familyExpiresAt := refreshTokenTestTime.Add(time.Hour)
	parent := createTestRefreshTokenFamily(ctx, t, db, user.ID, "hash-double-parent-"+suffix, familyExpiresAt)

	// Given: 트랜잭션 A가 P를 회전하고 커밋하지 않은 채 P의 행 잠금을 보유
	txA := beginLockHoldingTx(ctx, t, db)
	firstChild, err := RotateRefreshToken(ctx, txA, parent.ID, []byte("hash-double-child-a-"+suffix), familyExpiresAt, refreshTokenTestTime)
	if err != nil || firstChild == nil {
		t.Fatalf("expected rotation in A to succeed, got %+v, %v", firstChild, err)
	}

	// When: 다른 연결 B에서 같은 P를 회전 (A의 잠금을 기다림)
	connB, pidB := openDedicatedConn(ctx, t, db)
	defer connB.Close()
	resultB := make(chan rotateResult, 1)
	doneB := make(chan struct{})
	go func() {
		defer close(doneB)
		token, err := RotateRefreshToken(ctx, connB, parent.ID, []byte("hash-double-child-b-"+suffix), familyExpiresAt, refreshTokenTestTime)
		resultB <- rotateResult{token: token, err: err}
	}()
	// 중간에 테스트가 중단되어도 A를 끝내 B가 풀려나고, B가 끝난 뒤에 연결을 닫습니다
	defer func() {
		txA.Rollback()
		<-doneB
	}()

	waitForLockWait(ctx, t, db, pidB)
	if err := txA.Commit(); err != nil {
		t.Fatalf("failed to commit transaction A: %v", err)
	}
	result := <-resultB

	// Then: B는 에러 없이 nil, nil이고 P의 후속 토큰은 하나뿐
	if result.err != nil || result.token != nil {
		t.Errorf("expected nil, nil from B; got %+v, %v", result.token, result.err)
	}
	if count := countChildRefreshTokens(ctx, t, db, parent.ID); count != 1 {
		t.Errorf("child count = %d, want 1", count)
	}
}

// TestRotateRefreshToken_ConcurrentRevokeAndRotate는 회전이 진행 중일 때 계열 폐기가 실행되면
// 폐기를 빠져나간 후속 토큰이 남더라도 그 토큰으로는 회전되지 않는지 테스트합니다
func TestRotateRefreshToken_ConcurrentRevokeAndRotate(t *testing.T) {
	ctx, db, suffix := setupRefreshTokenConcurrencyTest(t)

	// Given: 커밋된 첫 토큰 P
	user := createCommittedRefreshTokenTestUser(ctx, t, db, suffix)
	familyExpiresAt := refreshTokenTestTime.Add(time.Hour)
	parent := createTestRefreshTokenFamily(ctx, t, db, user.ID, "hash-race-parent-"+suffix, familyExpiresAt)

	// Given: 트랜잭션 A가 P를 회전해 자식 C를 만들고 커밋하지 않은 채 P의 행 잠금을 보유
	txA := beginLockHoldingTx(ctx, t, db)
	childHash := "hash-race-child-" + suffix
	child, err := RotateRefreshToken(ctx, txA, parent.ID, []byte(childHash), familyExpiresAt, refreshTokenTestTime)
	if err != nil || child == nil {
		t.Fatalf("expected rotation in A to succeed, got %+v, %v", child, err)
	}

	// When: 다른 연결 B에서 계열을 폐기 (A의 잠금을 기다림)
	connB, pidB := openDedicatedConn(ctx, t, db)
	defer connB.Close()
	revokeErrB := make(chan error, 1)
	doneB := make(chan struct{})
	go func() {
		defer close(doneB)
		revokeErrB <- RevokeRefreshTokenFamily(ctx, connB, parent.FamilyID, models.RefreshTokenRevokedByReuse, refreshTokenTestTime)
	}()
	// 중간에 테스트가 중단되어도 A를 끝내 B가 풀려나고, B가 끝난 뒤에 연결을 닫습니다
	defer func() {
		txA.Rollback()
		<-doneB
	}()

	waitForLockWait(ctx, t, db, pidB)
	if err := txA.Commit(); err != nil {
		t.Fatalf("failed to commit transaction A: %v", err)
	}
	if err := <-revokeErrB; err != nil {
		t.Fatalf("expected revoke in B to succeed, got: %v", err)
	}

	// Then: 폐기 문장 시작 시점에 보이지 않던 C는 폐기되지 않은 채 남음 (경합이 재현됨)
	reloadedChild := mustGetRefreshToken(ctx, t, db, childHash)
	if reloadedChild.RevokedAt != nil {
		t.Fatalf("expected child to escape revocation, got RevokedAt %v", reloadedChild.RevokedAt)
	}

	// Then: 남은 C로는 회전되지 않고, 계열은 폐기된 것으로 판정됨
	grandchild, err := RotateRefreshToken(ctx, db, child.ID, []byte("hash-race-grandchild-"+suffix), familyExpiresAt, refreshTokenTestTime)
	if err != nil || grandchild != nil {
		t.Errorf("expected nil, nil when rotating escaped child; got %+v, %v", grandchild, err)
	}
	revoked, err := IsRefreshTokenFamilyRevoked(ctx, db, parent.FamilyID)
	if err != nil || !revoked {
		t.Errorf("expected family to be revoked; got %v, %v", revoked, err)
	}
}
