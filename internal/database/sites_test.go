package database

import (
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
		isOwner, err := IsUserSiteOwner(ctx, tx, user.ID, site.ID)
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
