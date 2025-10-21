package database

import (
	"fmt"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/june20516/orbithall/internal/testhelpers"
)

// TestCreateComment는 CreateComment 메서드를 테스트합니다
func TestCreateComment(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	site := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true)
	siteID := site.ID

	t.Run("최상위 댓글 생성 성공", func(t *testing.T) {
		// Given: 포스트가 존재함

		// posts 테이블에 테스트 포스트 삽입
		post := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post", "Test Post")
		postID := post.ID

		// When: CreateComment 호출
		authorName := "Test Author"
		password := "testpass123"
		content := "Test comment content"
		ipAddress := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		comment, err := CreateComment(ctx, tx, postID, nil, authorName, password, content, ipAddress, userAgent)

		// Then: 댓글 생성 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if comment == nil {
			t.Fatal("expected comment, got nil")
		}
		if comment.ID == 0 {
			t.Error("expected non-zero id")
		}
		if comment.PostID != postID {
			t.Errorf("expected post_id=%d, got %d", postID, comment.PostID)
		}
		if comment.ParentID != nil {
			t.Errorf("expected parent_id=nil, got %v", *comment.ParentID)
		}
		if comment.AuthorName != authorName {
			t.Errorf("expected author_name=%s, got %s", authorName, comment.AuthorName)
		}
		if comment.Content != content {
			t.Errorf("expected content=%s, got %s", content, comment.Content)
		}
		if comment.IPAddress != ipAddress {
			t.Errorf("expected ip_address=%s, got %s", ipAddress, comment.IPAddress)
		}
		if comment.UserAgent != userAgent {
			t.Errorf("expected user_agent=%s, got %s", userAgent, comment.UserAgent)
		}
		if comment.IsDeleted {
			t.Error("expected is_deleted=false, got true")
		}

		// 비밀번호가 bcrypt로 해싱되었는지 확인
		err = bcrypt.CompareHashAndPassword([]byte(comment.AuthorPassword), []byte(password))
		if err != nil {
			t.Errorf("password not hashed correctly: %v", err)
		}
	})

	t.Run("대댓글 생성 성공 (1depth)", func(t *testing.T) {
		// Given: 부모 댓글이 존재함
		var parentCommentID int64

		// posts 테이블에 테스트 포스트 삽입
		post := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-reply", "Test Post")
		postID := post.ID

		// 부모 댓글 삽입
		err := tx.QueryRowContext(ctx, `
			INSERT INTO comments (post_id, author_name, author_password, content, ip_address, user_agent, is_deleted)
			VALUES ($1, 'Parent', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIRSk8nHGC', 'Parent content', '10.0.0.1', 'Agent', FALSE)
			RETURNING id
		`, postID).Scan(&parentCommentID)
		if err != nil {
			t.Fatalf("failed to insert parent comment: %v", err)
		}

		// When: 대댓글 생성
		comment, err := CreateComment(ctx, tx, postID, &parentCommentID, "Reply Author", "replypass", "Reply content", "10.0.0.2", "Agent")

		// Then: 대댓글 생성 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if comment.ParentID == nil {
			t.Fatal("expected parent_id to be set, got nil")
		}
		if *comment.ParentID != parentCommentID {
			t.Errorf("expected parent_id=%d, got %d", parentCommentID, *comment.ParentID)
		}
	})

	t.Run("대댓글의 대댓글 생성 실패 (2depth 금지)", func(t *testing.T) {
		// Given: 대댓글이 존재함
		var parentCommentID int64
		var replyCommentID int64

		// posts 테이블에 테스트 포스트 삽입
		post := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-depth", "Test Post")
		postID := post.ID

		// 부모 댓글 삽입
		err := tx.QueryRowContext(ctx, `
			INSERT INTO comments (post_id, author_name, author_password, content, ip_address, user_agent, is_deleted)
			VALUES ($1, 'Parent', 'pass', 'Parent', '10.0.0.1', 'Agent', FALSE)
			RETURNING id
		`, postID).Scan(&parentCommentID)
		if err != nil {
			t.Fatalf("failed to insert test post: %v", err)
		}

		// 대댓글 삽입
		err = tx.QueryRowContext(ctx, `
			INSERT INTO comments (post_id, parent_id, author_name, author_password, content, ip_address, user_agent, is_deleted)
			VALUES ($1, $2, 'Reply', 'pass', 'Reply', '10.0.0.2', 'Agent', FALSE)
			RETURNING id
		`, postID, parentCommentID).Scan(&replyCommentID)
		if err != nil {
			t.Fatalf("failed to insert reply comment: %v", err)
		}

		// When: 대댓글의 대댓글 생성 시도
		_, err = CreateComment(ctx, tx, postID, &replyCommentID, "Nested Reply", "pass", "Nested", "10.0.0.3", "Agent")

		// Then: 에러 반환 (2depth 금지)
		if err == nil {
			t.Fatal("expected error for nested reply (2depth), got nil")
		}
	})

	t.Run("존재하지 않는 부모 댓글 ID로 생성 실패", func(t *testing.T) {
		// Given: 포스트는 존재하지만 부모 댓글은 없음
		var nonExistentParentID int64 = 99999

		// posts 테이블에 테스트 포스트 삽입
		post := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-invalid-parent", "Test Post")
		postID := post.ID

		_, err := CreateComment(ctx, tx, postID, &nonExistentParentID, "Author", "pass", "Content", "10.0.0.1", "Agent")

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for non-existent parent_id, got nil")
		}
	})
}

// TestGetCommentByID는 GetCommentByID 메서드를 테스트합니다
func TestGetCommentByID(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	site := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true)
	siteID := site.ID
	t.Run("존재하는 댓글 조회 성공", func(t *testing.T) {
		// Given: 댓글이 존재함

		// posts 테이블에 테스트 포스트 삽입
		post := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post", "Test Post")
		postID := post.ID

		// 댓글 생성
		comment, err := CreateComment(ctx, tx, postID, nil, "Test Author", "password123", "Test content", "192.168.1.1", "Mozilla/5.0")
		if err != nil {
			t.Fatalf("failed to create comment: %v", err)
		}

		// When: GetCommentByID 호출
		retrieved, err := GetCommentByID(ctx, tx, comment.ID)

		// Then: 댓글 조회 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if retrieved == nil {
			t.Fatal("expected comment, got nil")
		}
		if retrieved.ID != comment.ID {
			t.Errorf("expected id=%d, got %d", comment.ID, retrieved.ID)
		}
		if retrieved.AuthorName != "Test Author" {
			t.Errorf("expected author_name=Test Author, got %s", retrieved.AuthorName)
		}
		if retrieved.Content != "Test content" {
			t.Errorf("expected content=Test content, got %s", retrieved.Content)
		}
		if retrieved.IsDeleted {
			t.Error("expected is_deleted=false, got true")
		}
	})

	t.Run("존재하지 않는 댓글 조회 시 nil 반환", func(t *testing.T) {
		// Given: 존재하지 않는 댓글 ID
		var nonExistentID int64 = 99999

		// When: GetCommentByID 호출
		comment, err := GetCommentByID(ctx, tx, nonExistentID)

		// Then: nil 반환, 에러 없음
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if comment != nil {
			t.Errorf("expected nil comment, got: %+v", comment)
		}
	})

	t.Run("삭제된 댓글도 조회됨 (is_deleted=true)", func(t *testing.T) {
		// Given: 삭제된 댓글이 존재함

		// posts 테이블에 테스트 포스트 삽입
		post := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-deleted", "Test Post")
		postID := post.ID

		// 삭제된 댓글 삽입
		var commentID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO comments (post_id, author_name, author_password, content, ip_address, user_agent, is_deleted)
			VALUES ($1, 'Deleted Author', 'pass', 'Deleted content', '10.0.0.1', 'Agent', TRUE)
			RETURNING id
		`, postID).Scan(&commentID)
		if err != nil {
			t.Fatalf("failed to insert test post: %v", err)
		}

		// When: GetCommentByID 호출
		comment, err := GetCommentByID(ctx, tx, commentID)

		// Then: 삭제된 댓글도 조회됨
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if comment == nil {
			t.Fatal("expected comment, got nil")
		}
		if !comment.IsDeleted {
			t.Error("expected is_deleted=true, got false")
		}
	})
}

// TestUpdateComment는 UpdateComment 메서드를 테스트합니다
func TestUpdateComment(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	site := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true)
	siteID := site.ID

	t.Run("댓글 수정 성공", func(t *testing.T) {
		// Given: 댓글이 존재함

		// posts 테이블에 테스트 포스트 삽입
		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-update", "Test Post").ID

		// 댓글 생성
		originalContent := "Original content"
		originalIP := "192.168.1.1"
		originalUA := "Mozilla/5.0"
		comment, err := CreateComment(ctx, tx, postID, nil, "Test Author", "password123", originalContent, originalIP, originalUA)
		if err != nil {
			t.Fatalf("failed to create comment: %v", err)
		}

		// When: 댓글 수정 (다른 IP/User-Agent에서)
		newContent := "Updated content"
		newIP := "10.0.0.1"
		newUA := "Chrome/120.0"
		err = UpdateComment(ctx, tx, comment.ID, newContent, newIP, newUA)

		// Then: 수정 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// 수정된 댓글 조회하여 검증
		updated, err := GetCommentByID(ctx, tx, comment.ID)
		if err != nil {
			t.Fatalf("failed to get updated comment: %v", err)
		}
		if updated.Content != newContent {
			t.Errorf("expected content=%s, got %s", newContent, updated.Content)
		}
		// IP와 User-Agent가 수정 시점의 값으로 업데이트되었는지 확인
		if updated.IPAddress != newIP {
			t.Errorf("expected ip_address=%s, got %s", newIP, updated.IPAddress)
		}
		if updated.UserAgent != newUA {
			t.Errorf("expected user_agent=%s, got %s", newUA, updated.UserAgent)
		}
		// updated_at이 변경되었는지 확인
		if updated.UpdatedAt.Equal(updated.CreatedAt) {
			t.Error("expected updated_at to be different from created_at")
		}
	})

	t.Run("존재하지 않는 댓글 수정 시 에러 반환", func(t *testing.T) {
		// Given: 존재하지 않는 댓글 ID
		var nonExistentID int64 = 99999

		// When: 수정 시도
		err := UpdateComment(ctx, tx, nonExistentID, "New content", "10.0.0.1", "Agent")

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for non-existent comment, got nil")
		}
	})

	t.Run("삭제된 댓글 수정 시 에러 반환", func(t *testing.T) {
		// Given: 삭제된 댓글이 존재함

		var commentID int64

		// posts 테이블에 테스트 포스트 삽입
		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-deleted-update", "Test Post").ID

		// 삭제된 댓글 삽입
		err := tx.QueryRowContext(ctx, `
			INSERT INTO comments (post_id, author_name, author_password, content, ip_address, user_agent, is_deleted)
			VALUES ($1, 'Author', 'pass', 'Content', '10.0.0.1', 'Agent', TRUE)
			RETURNING id
		`, postID).Scan(&commentID)
		if err != nil {
			t.Fatalf("failed to insert deleted comment: %v", err)
		}

		// When: 삭제된 댓글 수정 시도
		err = UpdateComment(ctx, tx, commentID, "New content", "10.0.0.2", "Agent2")

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for deleted comment, got nil")
		}
	})
}

// TestDeleteComment는 DeleteComment 메서드를 테스트합니다
func TestDeleteComment(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	siteID := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true).ID

	t.Run("댓글 삭제 성공 (soft delete)", func(t *testing.T) {
		// Given: 댓글이 존재함

		// posts 테이블에 테스트 포스트 삽입
		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-delete", "Test Post").ID

		// 댓글 생성
		comment, err := CreateComment(ctx, tx, postID, nil, "Test Author", "password123", "Test content", "192.168.1.1", "Mozilla/5.0")
		if err != nil {
			t.Fatalf("failed to create comment: %v", err)
		}

		// When: 댓글 삭제
		err = DeleteComment(ctx, tx, comment.ID)

		// Then: 삭제 성공
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// 삭제된 댓글 조회하여 검증
		deleted, err := GetCommentByID(ctx, tx, comment.ID)
		if err != nil {
			t.Fatalf("failed to get deleted comment: %v", err)
		}
		if deleted == nil {
			t.Fatal("expected deleted comment, got nil")
		}
		if !deleted.IsDeleted {
			t.Error("expected is_deleted=true, got false")
		}
		if deleted.DeletedAt == nil {
			t.Error("expected deleted_at to be set, got nil")
		}
	})

	t.Run("존재하지 않는 댓글 삭제 시 에러 반환", func(t *testing.T) {
		// Given: 존재하지 않는 댓글 ID
		var nonExistentID int64 = 99999

		// When: 삭제 시도
		err := DeleteComment(ctx, tx, nonExistentID)

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for non-existent comment, got nil")
		}
	})

	t.Run("이미 삭제된 댓글 삭제 시 에러 반환", func(t *testing.T) {
		// Given: 이미 삭제된 댓글이 존재함

		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-delete", "Test Post").ID

		// 이미 삭제된 댓글 삽입
		var commentID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO comments (post_id, author_name, author_password, content, ip_address, user_agent, is_deleted, deleted_at)
			VALUES ($1, 'Author', 'pass', 'Content', '10.0.0.1', 'Agent', TRUE, NOW())
			RETURNING id
		`, postID).Scan(&commentID)
		if err != nil {
			t.Fatalf("failed to insert deleted comment: %v", err)
		}

		// When: 이미 삭제된 댓글 삭제 시도
		err = DeleteComment(ctx, tx, commentID)

		// Then: 에러 반환
		if err == nil {
			t.Fatal("expected error for already deleted comment, got nil")
		}
	})
}

// TestListComments는 ListComments 메서드를 테스트합니다
func TestListComments(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
	defer cleanup()

	siteID := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "test.com", []string{"http://localhost:3000"}, true).ID

	t.Run("댓글 목록 조회 성공 (2-level 계층 구조)", func(t *testing.T) {
		// Given: 포스트에 최상위 댓글 2개와 각각의 대댓글이 존재함

		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-list", "Test Post").ID

		// 최상위 댓글 1 생성
		comment1, err := CreateComment(ctx, tx, postID, nil, "Author1", "pass1", "Comment 1", "192.168.1.1", "Agent1")
		if err != nil {
			t.Fatalf("failed to create comment1: %v", err)
		}

		// 최상위 댓글 2 생성
		comment2, err := CreateComment(ctx, tx, postID, nil, "Author2", "pass2", "Comment 2", "192.168.1.2", "Agent2")
		if err != nil {
			t.Fatalf("failed to create comment2: %v", err)
		}

		// 댓글 1의 대댓글 생성
		_, err = CreateComment(ctx, tx, postID, &comment1.ID, "Reply1", "pass3", "Reply to 1", "10.0.0.1", "Agent3")
		if err != nil {
			t.Fatalf("failed to create reply1: %v", err)
		}

		// 댓글 2의 대댓글 생성
		_, err = CreateComment(ctx, tx, postID, &comment2.ID, "Reply2", "pass4", "Reply to 2", "10.0.0.2", "Agent4")
		if err != nil {
			t.Fatalf("failed to create reply2: %v", err)
		}

		// When: 댓글 목록 조회 (limit=10, offset=0)
		comments, total, err := ListComments(ctx, tx, postID, 10, 0)

		// Then: 2개의 최상위 댓글과 각각의 대댓글이 조회됨
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(comments) != 2 {
			t.Fatalf("expected 2 top-level comments, got %d", len(comments))
		}
		if total != 2 {
			t.Errorf("expected total=2, got %d", total)
		}

		// 첫 번째 최상위 댓글 검증 (created_at ASC 정렬이므로 comment1이 먼저)
		if comments[0].ID != comment1.ID {
			t.Errorf("expected first comment id=%d, got %d", comment1.ID, comments[0].ID)
		}
		if len(comments[0].Replies) != 1 {
			t.Errorf("expected 1 reply for comment1, got %d", len(comments[0].Replies))
		}
		if comments[0].Replies[0].Content != "Reply to 1" {
			t.Errorf("expected reply content='Reply to 1', got %s", comments[0].Replies[0].Content)
		}

		// 두 번째 최상위 댓글 검증
		if comments[1].ID != comment2.ID {
			t.Errorf("expected second comment id=%d, got %d", comment2.ID, comments[1].ID)
		}
		if len(comments[1].Replies) != 1 {
			t.Errorf("expected 1 reply for comment2, got %d", len(comments[1].Replies))
		}
	})

	t.Run("페이지네이션 동작 확인", func(t *testing.T) {
		// Given: 5개의 최상위 댓글이 존재함

		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-pagination", "Test Post").ID

		// 5개의 최상위 댓글 생성
		for i := 1; i <= 5; i++ {
			_, err := CreateComment(ctx, tx, postID, nil, fmt.Sprintf("Author%d", i), "pass", fmt.Sprintf("Comment %d", i), "192.168.1.1", "Agent")
			if err != nil {
				t.Fatalf("failed to create comment %d: %v", i, err)
			}
		}

		// When: limit=2, offset=0 (첫 페이지)
		page1, total1, err := ListComments(ctx, tx, postID, 2, 0)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(page1) != 2 {
			t.Errorf("expected 2 comments in page1, got %d", len(page1))
		}
		if total1 != 5 {
			t.Errorf("expected total=5, got %d", total1)
		}

		// When: limit=2, offset=2 (두 번째 페이지)
		page2, total2, err := ListComments(ctx, tx, postID, 2, 2)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(page2) != 2 {
			t.Errorf("expected 2 comments in page2, got %d", len(page2))
		}
		if total2 != 5 {
			t.Errorf("expected total=5, got %d", total2)
		}

		// When: limit=2, offset=4 (마지막 페이지)
		page3, total3, err := ListComments(ctx, tx, postID, 2, 4)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(page3) != 1 {
			t.Errorf("expected 1 comment in page3, got %d", len(page3))
		}
		if total3 != 5 {
			t.Errorf("expected total=5, got %d", total3)
		}
	})

	t.Run("댓글이 없는 포스트 조회", func(t *testing.T) {
		// Given: 댓글이 없는 포스트

		postID := testhelpers.CreateTestPost(ctx, t, tx, siteID, "test-post-empty", "Test Post").ID

		// When: 댓글 목록 조회
		comments, total, err := ListComments(ctx, tx, postID, 10, 0)

		// Then: 빈 목록 반환
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(comments) != 0 {
			t.Errorf("expected 0 comments, got %d", len(comments))
		}
		if total != 0 {
			t.Errorf("expected total=0, got %d", total)
		}
	})
}
