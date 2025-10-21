package database

import (
	// 1. context 임포트

	"testing"

	"github.com/june20516/orbithall/internal/testhelpers"
	// "github.com/lib/pq" // (pq는 insertTestSite 헬퍼에서 필요할 수 있음)
)

// TestGetPostBySlug는 GetPostBySlug 메서드를 테스트합니다
func TestGetPostBySlug(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	site := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true)
	siteID := site.ID
	slug := "test-post"
	title := "Test Post"

	t.Run("존재하는 포스트 조회 성공", func(t *testing.T) {
		// Given: 테스트 데이터 삽입

		// When: GetPostBySlug 호출
		post, err := GetPostBySlug(ctx, tx, siteID, slug)

		// Then: 포스트 조회 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if post == nil {
			t.Fatal("expected post, got nil")
		}
		if post.SiteID != siteID {
			t.Errorf("expected site_id=%d, got %d", siteID, post.SiteID)
		}
		if post.Slug != slug {
			t.Errorf("expected slug=%s, got %s", slug, post.Slug)
		}
		if post.Title != title {
			t.Errorf("expected title=%s, got %s", title, post.Title)
		}
	})

	t.Run("존재하지 않는 포스트 조회 시 nil 반환", func(t *testing.T) {

		// Given: 존재하지 않는 slug
		nonExistentSlug := "non-existent-post"

		// When: GetPostBySlug 호출
		post, err := GetPostBySlug(ctx, tx, siteID, nonExistentSlug)

		// Then: nil 반환, 에러 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if post != nil {
			t.Errorf("expected nil post, got: %+v", post)
		}
	})

	t.Run("다른 사이트의 포스트는 조회되지 않음", func(t *testing.T) {
		// Given: siteID에 포스트 삽입
		site2 := testhelpers.CreateTestSite(ctx, t, tx, "Test Site 2", "test2.com", []string{"http://localhost:3000"}, true)
		slug := "isolated-post"

		testhelpers.CreateTestPost(ctx, t, tx, siteID, slug, "Isolated Post")

		// When: site_id=2로 조회
		post, err := GetPostBySlug(ctx, tx, site2.ID, slug)

		// Then: nil 반환 (사이트 격리)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if post != nil {
			t.Errorf("expected nil post due to site isolation, got: %+v", post)
		}
	})
}

// TestGetPostByID는 GetPostByID 메서드를 테스트합니다
func TestGetPostByID(t *testing.T) {
	db := setupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	siteID := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true).ID

	t.Run("존재하는 포스트 조회 성공", func(t *testing.T) {
		// Given: 테스트 데이터 삽입
		slug := "test-post-by-id"
		title := "Test Post By ID"

		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, slug, title).ID

		// When: GetPostByID 호출
		post, err := GetPostByID(ctx, tx, postID)

		// Then: 포스트 조회 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if post == nil {
			t.Fatal("expected post, got nil")
		}
		if post.ID != postID {
			t.Errorf("expected id=%d, got %d", postID, post.ID)
		}
		if post.SiteID != siteID {
			t.Errorf("expected site_id=%d, got %d", siteID, post.SiteID)
		}
		if post.Slug != slug {
			t.Errorf("expected slug=%s, got %s", slug, post.Slug)
		}
		if post.Title != title {
			t.Errorf("expected title=%s, got %s", title, post.Title)
		}
	})

	t.Run("존재하지 않는 포스트 조회 시 nil 반환", func(t *testing.T) {
		// Given: 존재하지 않는 ID
		var nonExistentID int64 = 99999

		// When: GetPostByID 호출
		post, err := GetPostByID(ctx, tx, nonExistentID)

		// Then: nil 반환, 에러 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if post != nil {
			t.Errorf("expected nil post, got: %+v", post)
		}
	})
}

// TestGetOrCreatePost는 GetOrCreatePost 메서드를 테스트합니다
func TestGetOrCreatePost(t *testing.T) {
	db := setupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	siteID := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true).ID

	t.Run("존재하는 포스트는 조회만 함", func(t *testing.T) {
		// Given: 이미 존재하는 포스트
		slug := "existing-post"
		title := "Existing Post"

		// 기존 포스트 삽입
		existingID := testhelpers.CreateTestPost(ctx, t, tx, siteID, slug, title).ID

		// When: GetOrCreatePost 호출
		post, err := GetOrCreatePost(ctx, tx, siteID, slug, "New Title")

		// Then: 기존 포스트 반환 (title은 변경되지 않음)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if post == nil {
			t.Fatal("expected post, got nil")
		}
		if post.ID != existingID {
			t.Errorf("expected existing id=%d, got %d", existingID, post.ID)
		}
		if post.Title != title {
			t.Errorf("expected original title=%s, got %s", title, post.Title)
		}
	})

	t.Run("존재하지 않는 포스트는 생성함", func(t *testing.T) {
		// Given: 존재하지 않는 slug
		slug := "new-post-not-exist"
		title := "New Post"

		// When: GetOrCreatePost 호출
		post, err := GetOrCreatePost(ctx, tx, siteID, slug, title)

		// Then: 새로운 포스트 생성
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if post == nil {
			t.Fatal("expected post, got nil")
		}
		if post.ID == 0 {
			t.Error("expected non-zero id for new post")
		}
		if post.SiteID != siteID {
			t.Errorf("expected site_id=%d, got %d", siteID, post.SiteID)
		}
		if post.Slug != slug {
			t.Errorf("expected slug=%s, got %s", slug, post.Slug)
		}
		if post.Title != title {
			t.Errorf("expected title=%s, got %s", title, post.Title)
		}
		if post.CommentCount != 0 {
			t.Errorf("expected comment_count=0 for new post, got %d", post.CommentCount)
		}

		// 실제로 DB에 생성되었는지 확인
		var count int
		err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts WHERE site_id = $1 AND slug = $2", siteID, slug).Scan(&count)
		if err != nil {
			t.Fatalf("failed to verify post creation: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 post in db, got %d", count)
		}
	})

	t.Run("같은 slug를 여러 번 호출해도 하나만 생성됨", func(t *testing.T) {
		// Given: 새로운 slug
		slug := "idempotent-post"
		title := "Idempotent Post"

		// When: GetOrCreatePost를 3번 호출
		post1, err1 := GetOrCreatePost(ctx, tx, siteID, slug, title)
		post2, err2 := GetOrCreatePost(ctx, tx, siteID, slug, title)
		post3, err3 := GetOrCreatePost(ctx, tx, siteID, slug, title)

		// Then: 모두 같은 포스트 반환
		if err1 != nil || err2 != nil || err3 != nil {
			t.Fatalf("expected no errors, got: %v, %v, %v", err1, err2, err3)
		}
		if post1.ID != post2.ID || post2.ID != post3.ID {
			t.Errorf("expected same post id, got %d, %d, %d", post1.ID, post2.ID, post3.ID)
		}

		// DB에 하나만 존재하는지 확인
		var count int
		err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM posts WHERE site_id = $1 AND slug = $2", siteID, slug).Scan(&count)
		if err != nil {
			t.Fatalf("failed to verify post count: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 post in db, got %d", count)
		}
	})
}

// TestIncrementCommentCount는 IncrementCommentCount 메서드를 테스트합니다
func TestIncrementCommentCount(t *testing.T) {
	db := setupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	siteID := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true).ID

	t.Run("댓글 수 증가 성공", func(t *testing.T) {
		// Given: comment_count가 5인 포스트
		slug := "test-increment"
		initialCount := 5

		// 테스트 포스트 삽입
		var postID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO posts (site_id, slug, title, comment_count)
			VALUES ($1, $2, 'Test Post', $3)
			RETURNING id
		`, siteID, slug, initialCount).Scan(&postID)
		if err != nil {
			t.Fatalf("failed to insert test post: %v", err)
		}

		// When: IncrementCommentCount 호출
		err = IncrementCommentCount(ctx, tx, postID)

		// Then: 에러 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// comment_count가 1 증가했는지 확인
		var currentCount int
		err = tx.QueryRowContext(ctx, "SELECT comment_count FROM posts WHERE id = $1", postID).Scan(&currentCount)
		if err != nil {
			t.Fatalf("failed to get comment count: %v", err)
		}
		if currentCount != initialCount+1 {
			t.Errorf("expected comment_count=%d, got %d", initialCount+1, currentCount)
		}
	})

	t.Run("여러 번 호출 시 누적 증가", func(t *testing.T) {
		// Given: comment_count가 0인 포스트
		slug := "test-multiple-increment"

		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, slug, "Test Post").ID

		// When: IncrementCommentCount를 3번 호출
		for i := 0; i < 3; i++ {
			err := IncrementCommentCount(ctx, tx, postID)
			if err != nil {
				t.Fatalf("increment %d failed: %v", i+1, err)
			}
		}

		// Then: comment_count가 3이어야 함
		var currentCount int
		err := tx.QueryRowContext(ctx, "SELECT comment_count FROM posts WHERE id = $1", postID).Scan(&currentCount)
		if err != nil {
			t.Fatalf("failed to get comment count: %v", err)
		}
		if currentCount != 3 {
			t.Errorf("expected comment_count=3, got %d", currentCount)
		}
	})

	t.Run("존재하지 않는 포스트 ID는 에러 반환", func(t *testing.T) {
		// Given: 존재하지 않는 포스트 ID
		var nonExistentID int64 = 99999

		// When: IncrementCommentCount 호출
		err := IncrementCommentCount(ctx, tx, nonExistentID)

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for non-existent post, got nil")
		}
	})
}

// TestDecrementCommentCount는 DecrementCommentCount 메서드를 테스트합니다
func TestDecrementCommentCount(t *testing.T) {
	db := setupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	siteID := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true).ID

	t.Run("댓글 수 감소 성공", func(t *testing.T) {
		// Given: comment_count가 10인 포스트
		slug := "test-decrement"
		initialCount := 10

		// 테스트 포스트 삽입
		var postID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO posts (site_id, slug, title, comment_count)
			VALUES ($1, $2, 'Test Post', $3)
			RETURNING id
		`, siteID, slug, initialCount).Scan(&postID)
		if err != nil {
			t.Fatalf("failed to insert test post: %v", err)
		}

		// When: DecrementCommentCount 호출
		err = DecrementCommentCount(ctx, tx, postID)

		// Then: 에러 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// comment_count가 1 감소했는지 확인
		var currentCount int
		err = tx.QueryRowContext(ctx, "SELECT comment_count FROM posts WHERE id = $1", postID).Scan(&currentCount)
		if err != nil {
			t.Fatalf("failed to get comment count: %v", err)
		}
		if currentCount != initialCount-1 {
			t.Errorf("expected comment_count=%d, got %d", initialCount-1, currentCount)
		}
	})

	t.Run("0 이하로 내려가지 않음", func(t *testing.T) {
		// Given: comment_count가 0인 포스트
		slug := "test-zero-decrement"

		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, slug, "Test Post").ID

		// When: DecrementCommentCount 호출
		err := DecrementCommentCount(ctx, tx, postID)

		// Then: 에러 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// comment_count가 여전히 0이어야 함 (음수가 되지 않음)
		var currentCount int
		err = tx.QueryRowContext(ctx, "SELECT comment_count FROM posts WHERE id = $1", postID).Scan(&currentCount)
		if err != nil {
			t.Fatalf("failed to get comment count: %v", err)
		}
		if currentCount != 0 {
			t.Errorf("expected comment_count=0, got %d", currentCount)
		}
	})

	t.Run("여러 번 호출 시 누적 감소", func(t *testing.T) {
		// Given: comment_count가 5인 포스트
		slug := "test-multiple-decrement"

		var postID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO posts (site_id, slug, title, comment_count)
			VALUES ($1, $2, 'Test Post', 5)
			RETURNING id
		`, siteID, slug).Scan(&postID)
		if err != nil {
			t.Fatalf("failed to insert test post: %v", err)
		}

		// When: DecrementCommentCount를 3번 호출
		for i := 0; i < 3; i++ {
			err = DecrementCommentCount(ctx, tx, postID)
			if err != nil {
				t.Fatalf("decrement %d failed: %v", i+1, err)
			}
		}

		// Then: comment_count가 2여야 함 (5 - 3)
		var currentCount int
		err = tx.QueryRowContext(ctx, "SELECT comment_count FROM posts WHERE id = $1", postID).Scan(&currentCount)
		if err != nil {
			t.Fatalf("failed to get comment count: %v", err)
		}
		if currentCount != 2 {
			t.Errorf("expected comment_count=2, got %d", currentCount)
		}
	})

	t.Run("존재하지 않는 포스트 ID는 에러 반환", func(t *testing.T) {
		// Given: 존재하지 않는 포스트 ID
		var nonExistentID int64 = 99999

		// When: DecrementCommentCount 호출
		err := DecrementCommentCount(ctx, tx, nonExistentID)

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for non-existent post, got nil")
		}
	})
}
