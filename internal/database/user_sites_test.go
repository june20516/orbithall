package database

import (
	"testing"

	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
	"github.com/lib/pq"
)

// TestAddUserToSite는 사용자를 사이트에 연결하는 기능을 테스트합니다
func TestAddUserToSite(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("사용자-사이트 연결 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성
		user := &models.User{
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-123",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 테스트 사이트 생성
		site := testhelpers.CreateTestSite(ctx, t, tx, "Test Site", "example.com", []string{"https://example.com"}, true)

		// 사용자를 사이트에 연결
		err := AddUserToSite(ctx, tx, user.ID, site.ID, "owner")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// 연결 확인
		isOwner, err := IsUserSiteOwner(ctx, tx, user.ID, site.ID)
		if err != nil {
			t.Fatalf("Failed to check ownership: %v", err)
		}
		if !isOwner {
			t.Error("Expected user to be site owner")
		}
	})

	t.Run("중복 연결 방지 (복합 PK 위반)", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성
		user := &models.User{
			Email:    "test2@example.com",
			Name:     "Test User 2",
			GoogleID: "google-456",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 테스트 사이트 생성
		site := testhelpers.CreateTestSite(ctx, t, tx, "Test Site 2", "example2.com", []string{"https://example2.com"}, true)

		// 첫 번째 연결 (성공)
		err := AddUserToSite(ctx, tx, user.ID, site.ID, "owner")
		if err != nil {
			t.Fatalf("First connection should succeed: %v", err)
		}

		// 중복 연결 시도 (실패)
		err = AddUserToSite(ctx, tx, user.ID, site.ID, "owner")
		if err == nil {
			t.Error("Expected error for duplicate connection, got nil")
		}

		// PostgreSQL unique violation 에러 확인
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code != "23505" { // unique_violation
				t.Errorf("Expected unique_violation error code 23505, got %s", pqErr.Code)
			}
		}
	})
}

// TestGetUserSites는 사용자가 소유한 사이트 목록을 조회하는 기능을 테스트합니다
func TestGetUserSites(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("사용자의 사이트 목록 조회 (3개)", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성
		user := &models.User{
			Email:    "owner@example.com",
			Name:     "Site Owner",
			GoogleID: "google-owner",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 3개 사이트 생성 및 연결
		for i := 1; i <= 3; i++ {
			site := testhelpers.CreateTestSite(ctx, t, tx,
				"Site "+string(rune('0'+i)),
				"site"+string(rune('0'+i))+".com",
				[]string{"https://site" + string(rune('0'+i)) + ".com"},
				true,
			)
			if err := AddUserToSite(ctx, tx, user.ID, site.ID, "owner"); err != nil {
				t.Fatalf("Failed to add user to site: %v", err)
			}
		}

		// 사용자의 사이트 목록 조회
		sites, err := GetUserSites(ctx, tx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user sites: %v", err)
		}

		// 3개 사이트 확인
		if len(sites) != 3 {
			t.Errorf("Expected 3 sites, got %d", len(sites))
		}
	})

	t.Run("사이트가 없는 사용자는 빈 배열 반환", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성 (사이트 없음)
		user := &models.User{
			Email:    "nosite@example.com",
			Name:     "No Site User",
			GoogleID: "google-nosite",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 사용자의 사이트 목록 조회
		sites, err := GetUserSites(ctx, tx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user sites: %v", err)
		}

		// 빈 배열 확인
		if len(sites) != 0 {
			t.Errorf("Expected 0 sites, got %d", len(sites))
		}
	})
}

// TestGetSiteUsers는 사이트에 연결된 사용자 목록을 조회하는 기능을 테스트합니다
func TestGetSiteUsers(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("사이트의 사용자 목록 조회 (2명)", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사이트 생성
		site := testhelpers.CreateTestSite(ctx, t, tx, "Shared Site", "shared.com", []string{"https://shared.com"}, true)

		// 2명 사용자 생성 및 연결
		for i := 1; i <= 2; i++ {
			user := &models.User{
				Email:    "user" + string(rune('0'+i)) + "@example.com",
				Name:     "User " + string(rune('0'+i)),
				GoogleID: "google-user-" + string(rune('0'+i)),
			}
			if err := CreateUser(ctx, tx, user); err != nil {
				t.Fatalf("Failed to create user: %v", err)
			}
			if err := AddUserToSite(ctx, tx, user.ID, site.ID, "owner"); err != nil {
				t.Fatalf("Failed to add user to site: %v", err)
			}
		}

		// 사이트의 사용자 목록 조회
		users, err := GetSiteUsers(ctx, tx, site.ID)
		if err != nil {
			t.Fatalf("Failed to get site users: %v", err)
		}

		// 2명 확인
		if len(users) != 2 {
			t.Errorf("Expected 2 users, got %d", len(users))
		}
	})
}

// TestRemoveUserFromSite는 사용자-사이트 연결을 해제하는 기능을 테스트합니다
func TestRemoveUserFromSite(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("연결 해제 후 빈 목록 확인", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성
		user := &models.User{
			Email:    "remove@example.com",
			Name:     "Remove Test",
			GoogleID: "google-remove",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 테스트 사이트 생성 및 연결
		site := testhelpers.CreateTestSite(ctx, t, tx, "Remove Site", "remove.com", []string{"https://remove.com"}, true)
		if err := AddUserToSite(ctx, tx, user.ID, site.ID, "owner"); err != nil {
			t.Fatalf("Failed to add user to site: %v", err)
		}

		// 연결 해제
		err := RemoveUserFromSite(ctx, tx, user.ID, site.ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// 사이트 목록 조회 (빈 배열)
		sites, err := GetUserSites(ctx, tx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user sites: %v", err)
		}
		if len(sites) != 0 {
			t.Errorf("Expected 0 sites after removal, got %d", len(sites))
		}

		// 소유자 확인 (false)
		isOwner, err := IsUserSiteOwner(ctx, tx, user.ID, site.ID)
		if err != nil {
			t.Fatalf("Failed to check ownership: %v", err)
		}
		if isOwner {
			t.Error("Expected user to not be owner after removal")
		}
	})
}

// TestIsUserSiteOwner는 사용자가 사이트 소유자인지 확인하는 기능을 테스트합니다
func TestIsUserSiteOwner(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer Close(db)

	t.Run("소유자 확인 (true)", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성
		user := &models.User{
			Email:    "owner@example.com",
			Name:     "Owner",
			GoogleID: "google-owner-check",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 테스트 사이트 생성 및 연결
		site := testhelpers.CreateTestSite(ctx, t, tx, "Owner Site", "owner.com", []string{"https://owner.com"}, true)
		if err := AddUserToSite(ctx, tx, user.ID, site.ID, "owner"); err != nil {
			t.Fatalf("Failed to add user to site: %v", err)
		}

		// 소유자 확인
		isOwner, err := IsUserSiteOwner(ctx, tx, user.ID, site.ID)
		if err != nil {
			t.Fatalf("Failed to check ownership: %v", err)
		}
		if !isOwner {
			t.Error("Expected user to be owner")
		}
	})

	t.Run("소유자 아님 (false)", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// 테스트 사용자 생성
		user := &models.User{
			Email:    "notowner@example.com",
			Name:     "Not Owner",
			GoogleID: "google-not-owner",
		}
		if err := CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 테스트 사이트 생성 (연결 안 함)
		site := testhelpers.CreateTestSite(ctx, t, tx, "Not Owner Site", "notowner.com", []string{"https://notowner.com"}, true)

		// 소유자 확인 (false)
		isOwner, err := IsUserSiteOwner(ctx, tx, user.ID, site.ID)
		if err != nil {
			t.Fatalf("Failed to check ownership: %v", err)
		}
		if isOwner {
			t.Error("Expected user to not be owner")
		}
	})
}
