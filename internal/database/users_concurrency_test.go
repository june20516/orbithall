package database

import (
	"context"
	"testing"

	"github.com/june20516/orbithall/internal/models"
)

// getOrCreateUserResult는 goroutine에서 실행한 사용자 조회·생성의 결과입니다
type getOrCreateUserResult struct {
	user *models.User
	err  error
}

// TestGetOrCreateUserByGoogleID_ConcurrentFirstLogin은 같은 Google ID로 첫 로그인이 동시에 들어올 때
// 두 요청 모두 에러 없이 같은 사용자를 받는지 테스트합니다
func TestGetOrCreateUserByGoogleID_ConcurrentFirstLogin(t *testing.T) {
	ctx, db, suffix := setupConcurrencyTest(t)

	// Given: 아직 없는 사용자 정보, 테스트 종료 시 커밋된 사용자를 삭제하도록 등록
	// 삭제를 트랜잭션 A의 롤백보다 먼저 등록해, 정리 시에는 롤백이 먼저 실행되게 합니다
	input := models.User{
		Email:    suffix + "@example.com",
		Name:     "Concurrent First Login User",
		GoogleID: "google-" + suffix,
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DELETE FROM users WHERE google_id = $1`, input.GoogleID); err != nil {
			t.Errorf("failed to delete committed test user: %v", err)
		}
	})

	// Given: 트랜잭션 A가 사용자를 INSERT하고 커밋하지 않은 채 대기
	txA := beginLockHoldingTx(ctx, t, db)
	inputA := input
	userA, err := GetOrCreateUserByGoogleID(ctx, txA, &inputA)
	if err != nil {
		t.Fatalf("expected get-or-create in A to succeed, got: %v", err)
	}

	// When: 다른 연결 B에서 같은 Google ID로 조회·생성 (ON CONFLICT 판정이 A의 커밋 여부를 기다림)
	connB, pidB := openDedicatedConn(ctx, t, db)
	defer connB.Close()
	resultB := make(chan getOrCreateUserResult, 1)
	doneB := make(chan struct{})
	go func() {
		defer close(doneB)
		inputB := input
		user, err := GetOrCreateUserByGoogleID(ctx, connB, &inputB)
		resultB <- getOrCreateUserResult{user: user, err: err}
	}()
	// 중간에 테스트가 중단되어도 A를 끝내 B가 풀려나고, B가 끝난 뒤에 연결을 닫습니다
	defer func() {
		txA.Rollback()
		<-doneB
	}()

	waitForLockWait(ctx, t, db, pidB, "users")
	if err := txA.Commit(); err != nil {
		t.Fatalf("failed to commit transaction A: %v", err)
	}
	result := <-resultB

	// Then: B는 에러 없이 A가 만든 사용자를 받고, 행은 하나뿐
	if result.err != nil {
		t.Fatalf("expected no error from B, got: %v", result.err)
	}
	if result.user == nil || result.user.ID != userA.ID {
		t.Errorf("B user = %+v, want id %d", result.user, userA.ID)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE google_id = $1`, input.GoogleID).Scan(&count); err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 1 {
		t.Errorf("users rows = %d, want 1", count)
	}
}
