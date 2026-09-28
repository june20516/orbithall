package database

import (
	"context"
	"fmt"
	"sync"
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

// concurrentGetOrCreateWorkers는 같은 사용자를 동시에 조회·생성하는 goroutine 수입니다
const concurrentGetOrCreateWorkers = 16

// concurrentGetOrCreateRounds는 새 사용자로 동시 조회·생성을 반복하는 횟수입니다
// 경합은 확률적으로만 드러나므로 여러 번 반복해 재현 가능성을 높입니다
const concurrentGetOrCreateRounds = 50

// TestGetOrCreateUserByGoogleID_ConcurrentInsert는 트랜잭션 없이 같은 Google ID·이메일로
// 여러 요청이 동시에 INSERT를 시도해도 에러 없이 모두 같은 사용자를 받는지 테스트합니다
// users에는 google_id 외에 email UNIQUE도 있어, 두 제약 모두에서 경합이 에러로 드러나지 않아야 합니다
func TestGetOrCreateUserByGoogleID_ConcurrentInsert(t *testing.T) {
	ctx, db, suffix := setupConcurrencyTest(t)

	// Given: 라운드마다 쓰는 고유 Google ID 접두어, 테스트 종료 시 커밋된 사용자를 모두 삭제
	googleIDPrefix := "google-" + suffix + "-"
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), `DELETE FROM users WHERE google_id LIKE $1 || '%'`, googleIDPrefix); err != nil {
			t.Errorf("failed to delete committed test users: %v", err)
		}
	})

	for round := 0; round < concurrentGetOrCreateRounds; round++ {
		// Given: 아직 없는 사용자 정보
		input := models.User{
			Email:    fmt.Sprintf("%s-%d@example.com", suffix, round),
			Name:     "Concurrent Insert User",
			GoogleID: fmt.Sprintf("%s%d", googleIDPrefix, round),
		}

		// When: 여러 goroutine이 동시에 같은 사용자를 조회·생성 (시작 신호로 최대한 같은 순간에 출발)
		start := make(chan struct{})
		results := make([]getOrCreateUserResult, concurrentGetOrCreateWorkers)
		var wg sync.WaitGroup
		for i := range results {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				workerInput := input
				user, err := GetOrCreateUserByGoogleID(ctx, db, &workerInput)
				results[i] = getOrCreateUserResult{user: user, err: err}
			}(i)
		}
		close(start)
		wg.Wait()

		// Then: 모든 요청이 에러 없이 같은 ID의 사용자를 받음
		var firstID int64
		for i, result := range results {
			if result.err != nil {
				t.Fatalf("round %d worker %d: expected no error, got: %v", round, i, result.err)
			}
			if result.user == nil {
				t.Fatalf("round %d worker %d: expected user, got nil", round, i)
			}
			if firstID == 0 {
				firstID = result.user.ID
			}
			if result.user.ID != firstID {
				t.Fatalf("round %d worker %d: user id = %d, want %d", round, i, result.user.ID, firstID)
			}
		}
	}
}
