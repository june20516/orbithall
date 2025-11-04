package database

import (
	"database/sql"
	"testing"

	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

// TestCreateSiteForUser는 사이트 생성 및 사용자 연결 기능을 테스트합니다
func TestCreateSiteForUser(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("사이트 생성 및 사용자 자동 연결", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성
		user := &models.User{
			Email:    "siteowner@example.com",
			Name:     "Site Owner",
			GoogleID: "google-site-owner",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 사이트 생성
		site := &models.Site{
			Name:        "Test Site",
			Domain:      "testsite.com",
			CORSOrigins: []string{"https://testsite.com"},
			IsActive:    true,
		}

		err := CreateSiteForUser(ctx, tx, site, user.ID)
		if err != nil {
			t.Fatalf("Failed to create site for user: %v", err)
		}

		// Site ID가 설정되었는지 확인
		if site.ID == 0 {
			t.Error("Expected site ID to be set, got 0")
		}

		// API Key가 자동 생성되었는지 확인
		if site.APIKey == "" {
			t.Error("Expected API key to be generated")
		}
		if len(site.APIKey) < 10 {
			t.Errorf("Expected API key length > 10, got %d", len(site.APIKey))
		}

		// 사용자의 사이트 목록에 포함되는지 확인
		sites, err := GetUserSites(ctx, tx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user sites: %v", err)
		}
		if len(sites) != 1 {
			t.Errorf("Expected 1 site, got %d", len(sites))
		}
		if len(sites) > 0 && sites[0].ID != site.ID {
			t.Errorf("Expected site ID %d, got %d", site.ID, sites[0].ID)
		}

		// 사용자가 소유자인지 확인
		isOwner, err := HasUserSiteAccess(ctx, tx, user.ID, site.ID)
		if err != nil {
			t.Fatalf("Failed to check ownership: %v", err)
		}
		if !isOwner {
			t.Error("Expected user to be site owner")
		}
	})

	t.Run("여러 사이트 생성 (3개)", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성
		user := &models.User{
			Email:    "multisiteowner@example.com",
			Name:     "Multi Site Owner",
			GoogleID: "google-multi-site",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 3개 사이트 생성
		for i := 1; i <= 3; i++ {
			site := &models.Site{
				Name:        "Site " + string(rune('0'+i)),
				Domain:      "site" + string(rune('0'+i)) + ".com",
				CORSOrigins: []string{"https://site" + string(rune('0'+i)) + ".com"},
				IsActive:    true,
			}

			if err := CreateSiteForUser(ctx, tx, site, user.ID); err != nil {
				t.Fatalf("Failed to create site %d: %v", i, err)
			}
		}

		// 사용자의 사이트 목록 확인 (3개)
		sites, err := GetUserSites(ctx, tx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user sites: %v", err)
		}
		if len(sites) != 3 {
			t.Errorf("Expected 3 sites, got %d", len(sites))
		}
	})

	t.Run("API Key 중복 방지", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성
		user := &models.User{
			Email:    "apikeytest@example.com",
			Name:     "API Key Test",
			GoogleID: "google-apikey-test",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 사이트 1 생성
		site1 := &models.Site{
			Name:        "Site 1",
			Domain:      "site1.com",
			CORSOrigins: []string{"https://site1.com"},
			IsActive:    true,
		}
		if err := CreateSiteForUser(ctx, tx, site1, user.ID); err != nil {
			t.Fatalf("Failed to create site 1: %v", err)
		}

		// 사이트 2 생성
		site2 := &models.Site{
			Name:        "Site 2",
			Domain:      "site2.com",
			CORSOrigins: []string{"https://site2.com"},
			IsActive:    true,
		}
		if err := CreateSiteForUser(ctx, tx, site2, user.ID); err != nil {
			t.Fatalf("Failed to create site 2: %v", err)
		}

		// API Key가 서로 다른지 확인
		if site1.APIKey == site2.APIKey {
			t.Error("Expected different API keys for different sites")
		}
	})
}

// TestGetSiteByID는 ID로 사이트 조회 기능을 테스트합니다
func TestGetSiteByID(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("사이트 조회 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 및 사이트 생성
		user := &models.User{
			Email:    "getsitetest@example.com",
			Name:     "Get Site Test",
			GoogleID: "google-getsite",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		site := &models.Site{
			Name:        "Test Site",
			Domain:      "testsite.com",
			CORSOrigins: []string{"https://testsite.com"},
			IsActive:    true,
		}
		if err := CreateSiteForUser(ctx, tx, site, user.ID); err != nil {
			t.Fatalf("Failed to create site: %v", err)
		}

		// 사이트 조회
		retrievedSite, err := GetSiteByID(ctx, tx, site.ID)
		if err != nil {
			t.Fatalf("Failed to get site: %v", err)
		}

		// 필드 검증
		if retrievedSite.ID != site.ID {
			t.Errorf("Expected ID %d, got %d", site.ID, retrievedSite.ID)
		}
		if retrievedSite.Name != site.Name {
			t.Errorf("Expected name %s, got %s", site.Name, retrievedSite.Name)
		}
		if retrievedSite.Domain != site.Domain {
			t.Errorf("Expected domain %s, got %s", site.Domain, retrievedSite.Domain)
		}
	})

	t.Run("존재하지 않는 사이트 조회", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 존재하지 않는 ID로 조회
		_, err := GetSiteByID(ctx, tx, 99999)
		if err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows, got %v", err)
		}
	})
}

// TestUpdateSite는 사이트 수정 기능을 테스트합니다
func TestUpdateSite(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("사이트 수정 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 및 사이트 생성
		user := &models.User{
			Email:    "updatetest@example.com",
			Name:     "Update Test",
			GoogleID: "google-update",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		site := &models.Site{
			Name:        "Original Name",
			Domain:      "original.com",
			CORSOrigins: []string{"https://original.com"},
			IsActive:    true,
		}
		if err := CreateSiteForUser(ctx, tx, site, user.ID); err != nil {
			t.Fatalf("Failed to create site: %v", err)
		}

		// 사이트 수정
		newName := "Updated Name"
		newCORSOrigins := []string{"https://updated.com", "http://localhost:3000"}
		newIsActive := false

		err := UpdateSite(ctx, tx, site.ID, newName, newCORSOrigins, newIsActive)
		if err != nil {
			t.Fatalf("Failed to update site: %v", err)
		}

		// 수정된 내용 확인
		updatedSite, err := GetSiteByID(ctx, tx, site.ID)
		if err != nil {
			t.Fatalf("Failed to get updated site: %v", err)
		}

		if updatedSite.Name != newName {
			t.Errorf("Expected name %s, got %s", newName, updatedSite.Name)
		}
		if len(updatedSite.CORSOrigins) != 2 {
			t.Errorf("Expected 2 CORS origins, got %d", len(updatedSite.CORSOrigins))
		}
		if updatedSite.IsActive != newIsActive {
			t.Errorf("Expected is_active %v, got %v", newIsActive, updatedSite.IsActive)
		}

		// domain과 api_key는 변경되지 않아야 함
		if updatedSite.Domain != site.Domain {
			t.Errorf("Domain should not change, got %s", updatedSite.Domain)
		}
		if updatedSite.APIKey != site.APIKey {
			t.Errorf("API key should not change")
		}
	})

	t.Run("존재하지 않는 사이트 수정", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 존재하지 않는 ID로 수정 시도
		err := UpdateSite(ctx, tx, 99999, "New Name", []string{"https://new.com"}, true)
		if err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows, got %v", err)
		}
	})
}

// TestDeleteSite는 사이트 삭제 기능을 테스트합니다
func TestDeleteSite(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("사이트 삭제 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 및 사이트 생성
		user := &models.User{
			Email:    "deletetest@example.com",
			Name:     "Delete Test",
			GoogleID: "google-delete",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		site := &models.Site{
			Name:        "Site to Delete",
			Domain:      "todelete.com",
			CORSOrigins: []string{"https://todelete.com"},
			IsActive:    true,
		}
		if err := CreateSiteForUser(ctx, tx, site, user.ID); err != nil {
			t.Fatalf("Failed to create site: %v", err)
		}

		// 사이트 삭제
		err := DeleteSite(ctx, tx, site.ID)
		if err != nil {
			t.Fatalf("Failed to delete site: %v", err)
		}

		// 삭제 확인 (조회 시 에러)
		_, err = GetSiteByID(ctx, tx, site.ID)
		if err != sql.ErrNoRows {
			t.Errorf("Expected site to be deleted, got error: %v", err)
		}
	})

	t.Run("존재하지 않는 사이트 삭제", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 존재하지 않는 ID로 삭제 시도
		err := DeleteSite(ctx, tx, 99999)
		if err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows, got %v", err)
		}
	})
}

// TestGetSiteStats는 사이트 통계 조회 기능을 테스트합니다
func TestGetSiteStats(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("통계 조회 성공 - Post와 댓글이 있는 경우", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사이트 생성
		site := testhelpers.CreateTestSite(ctx, t, tx, "Stats Test Site", "stats.test.com",
			[]string{"https://stats.test.com"}, true)

		// 3개의 다른 Post에 각각 3개의 댓글 생성
		posts := []string{"post-1", "post-2", "post-3"}
		totalComments := 0
		for _, postSlug := range posts {
			post := testhelpers.CreateTestPost(ctx, t, tx, site.ID, postSlug, "Test Post")

			for i := 0; i < 3; i++ {
				_, err := CreateComment(ctx, tx, post.ID, nil, "Author", "password123",
					"Test comment", "127.0.0.1", "test-agent")
				if err != nil {
					t.Fatalf("Failed to create comment: %v", err)
				}
				totalComments++
			}
		}

		// 통계 조회
		stats, err := GetSiteStats(ctx, tx, site.ID)
		if err != nil {
			t.Fatalf("Failed to get site stats: %v", err)
		}

		// 검증
		if stats.PostCount != 3 {
			t.Errorf("Expected post count 3, got %d", stats.PostCount)
		}
		if stats.CommentCount != totalComments {
			t.Errorf("Expected comment count %d, got %d", totalComments, stats.CommentCount)
		}
		if stats.DeletedCommentCount != 0 {
			t.Errorf("Expected deleted comment count 0, got %d", stats.DeletedCommentCount)
		}
	})

	t.Run("통계 조회 성공 - 삭제된 댓글 별도 카운트", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사이트 생성
		site := testhelpers.CreateTestSite(ctx, t, tx, "Stats Test Site 2", "stats2.test.com",
			[]string{"https://stats2.test.com"}, true)
		post := testhelpers.CreateTestPost(ctx, t, tx, site.ID, "post-1", "Test Post")

		// 정상 댓글 2개 생성
		for i := 0; i < 2; i++ {
			_, err := CreateComment(ctx, tx, post.ID, nil, "Author", "password123",
				"Test comment", "127.0.0.1", "test-agent")
			if err != nil {
				t.Fatalf("Failed to create comment: %v", err)
			}
		}

		// 댓글 1개를 삭제 처리
		var commentID int64
		err := tx.QueryRowContext(ctx, `
			SELECT id FROM comments WHERE post_id = $1 LIMIT 1
		`, post.ID).Scan(&commentID)
		if err != nil {
			t.Fatalf("Failed to get comment ID: %v", err)
		}

		err = DeleteComment(ctx, tx, commentID)
		if err != nil {
			t.Fatalf("Failed to delete comment: %v", err)
		}

		// 통계 조회
		stats, err := GetSiteStats(ctx, tx, site.ID)
		if err != nil {
			t.Fatalf("Failed to get site stats: %v", err)
		}

		// 검증 - 삭제된 댓글은 별도 카운트
		if stats.PostCount != 1 {
			t.Errorf("Expected post count 1, got %d", stats.PostCount)
		}
		if stats.CommentCount != 1 {
			t.Errorf("Expected comment count 1 (active comments), got %d", stats.CommentCount)
		}
		if stats.DeletedCommentCount != 1 {
			t.Errorf("Expected deleted comment count 1, got %d", stats.DeletedCommentCount)
		}
	})

	t.Run("통계 조회 성공 - 댓글이 없는 경우", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사이트 생성 (댓글 없음)
		site := testhelpers.CreateTestSite(ctx, t, tx, "Empty Stats Site", "empty.test.com",
			[]string{"https://empty.test.com"}, true)

		// 통계 조회
		stats, err := GetSiteStats(ctx, tx, site.ID)
		if err != nil {
			t.Fatalf("Failed to get site stats: %v", err)
		}

		// 검증
		if stats.PostCount != 0 {
			t.Errorf("Expected post count 0, got %d", stats.PostCount)
		}
		if stats.CommentCount != 0 {
			t.Errorf("Expected comment count 0, got %d", stats.CommentCount)
		}
		if stats.DeletedCommentCount != 0 {
			t.Errorf("Expected deleted comment count 0, got %d", stats.DeletedCommentCount)
		}
	})
}
