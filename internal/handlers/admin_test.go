package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

// TestListSites는 내 사이트 목록 조회 기능을 테스트합니다
func TestListSites(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("사이트 목록 조회 성공 - 3개", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자 생성
		user := &models.User{
			Email:    "admin@example.com",
			Name:     "Admin User",
			GoogleID: "google-admin",
		}
		if err := database.CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// 3개의 사이트 생성
		for i := 1; i <= 3; i++ {
			site := &models.Site{
				Name:        "Site " + strconv.Itoa(i),
				Domain:      "site" + strconv.Itoa(i) + ".com",
				CORSOrigins: []string{"https://site" + strconv.Itoa(i) + ".com"},
				IsActive:    true,
			}
			if err := database.CreateSiteForUser(ctx, tx, site, user.ID); err != nil {
				t.Fatalf("Failed to create site %d: %v", i, err)
			}
		}

		// When: ListSites 호출
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/sites", nil)
		req = req.WithContext(context.WithValue(ctx, userContextKey, user))
		rec := httptest.NewRecorder()

		handler.ListSites(rec, req)

		// Then: 200 OK, 3개 사이트 반환
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		sites, ok := response["sites"].([]interface{})
		if !ok {
			t.Fatal("Expected 'sites' field in response")
		}

		if len(sites) != 3 {
			t.Errorf("Expected 3 sites, got %d", len(sites))
		}
	})

	t.Run("사이트 목록 조회 성공 - 0개", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사이트가 없는 사용자
		user := &models.User{
			Email:    "nosite@example.com",
			Name:     "No Site User",
			GoogleID: "google-nosite",
		}
		if err := database.CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// When: ListSites 호출
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/sites", nil)
		req = req.WithContext(context.WithValue(ctx, userContextKey, user))
		rec := httptest.NewRecorder()

		handler.ListSites(rec, req)

		// Then: 200 OK, 빈 배열
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		sites := response["sites"].([]interface{})

		if len(sites) != 0 {
			t.Errorf("Expected 0 sites, got %d", len(sites))
		}
	})
}

// TestGetSite는 사이트 상세 조회 기능을 테스트합니다
func TestGetSite(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("사이트 조회 성공 - 소유자", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자와 사이트 생성
		user := &models.User{
			Email:    "owner@example.com",
			Name:     "Owner",
			GoogleID: "google-owner",
		}
		if err := database.CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		site := &models.Site{
			Name:        "My Site",
			Domain:      "mysite.com",
			CORSOrigins: []string{"https://mysite.com"},
			IsActive:    true,
		}
		if err := database.CreateSiteForUser(ctx, tx, site, user.ID); err != nil {
			t.Fatalf("Failed to create site: %v", err)
		}

		// When: GetSite 호출
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/sites/"+strconv.FormatInt(site.ID, 10), nil)
		req = req.WithContext(context.WithValue(ctx, userContextKey, user))

		// Chi URL 파라미터 설정
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()

		handler.GetSite(rec, req)

		// Then: 200 OK
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["name"] != site.Name {
			t.Errorf("Expected name %s, got %v", site.Name, response["name"])
		}
	})

	t.Run("사이트 조회 실패 - 소유하지 않음", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 두 사용자와 사이트 (user1이 소유)
		user1 := &models.User{
			Email:    "user1@example.com",
			Name:     "User 1",
			GoogleID: "google-user1",
		}
		database.CreateUser(ctx, tx, user1)

		user2 := &models.User{
			Email:    "user2@example.com",
			Name:     "User 2",
			GoogleID: "google-user2",
		}
		database.CreateUser(ctx, tx, user2)

		site := &models.Site{
			Name:        "User1 Site",
			Domain:      "user1site.com",
			CORSOrigins: []string{"https://user1site.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user1.ID)

		// When: user2가 user1의 사이트 조회 시도
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/sites/"+strconv.FormatInt(site.ID, 10), nil)
		req = req.WithContext(context.WithValue(ctx, userContextKey, user2))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()

		handler.GetSite(rec, req)

		// Then: 404 Not Found
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("사이트 조회 실패 - 존재하지 않는 ID", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		user := &models.User{
			Email:    "user@example.com",
			Name:     "User",
			GoogleID: "google-user",
		}
		database.CreateUser(ctx, tx, user)

		// When: 존재하지 않는 사이트 ID로 조회
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/sites/99999", nil)
		req = req.WithContext(context.WithValue(ctx, userContextKey, user))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "99999")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()

		handler.GetSite(rec, req)

		// Then: 404 Not Found
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

// TestCreateSite는 사이트 생성 기능을 테스트합니다
func TestCreateSite(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("사이트 생성 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자 생성
		user := &models.User{
			Email:    "creator@example.com",
			Name:     "Creator",
			GoogleID: "google-creator",
		}
		database.CreateUser(ctx, tx, user)

		// When: CreateSite 호출
		requestBody := map[string]interface{}{
			"name":         "New Site",
			"domain":       "newsite.com",
			"cors_origins": []string{"https://newsite.com"},
		}
		bodyBytes, _ := json.Marshal(requestBody)

		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodPost, "/admin/sites", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(ctx, userContextKey, user))
		rec := httptest.NewRecorder()

		handler.CreateSite(rec, req)

		// Then: 201 Created
		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["name"] != "New Site" {
			t.Errorf("Expected name 'New Site', got %v", response["name"])
		}
		if response["api_key"] == nil || response["api_key"] == "" {
			t.Error("Expected API key to be generated")
		}
	})

	t.Run("사이트 생성 실패 - 입력 검증 오류", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		user := &models.User{
			Email:    "creator2@example.com",
			Name:     "Creator 2",
			GoogleID: "google-creator2",
		}
		database.CreateUser(ctx, tx, user)

		// When: name 누락
		requestBody := map[string]interface{}{
			"domain":       "newsite.com",
			"cors_origins": []string{"https://newsite.com"},
		}
		bodyBytes, _ := json.Marshal(requestBody)

		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodPost, "/admin/sites", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(ctx, userContextKey, user))
		rec := httptest.NewRecorder()

		handler.CreateSite(rec, req)

		// Then: 400 Bad Request
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

// TestUpdateSite는 사이트 수정 기능을 테스트합니다
func TestUpdateSite(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("사이트 수정 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자와 사이트 생성
		user := &models.User{
			Email:    "updater@example.com",
			Name:     "Updater",
			GoogleID: "google-updater",
		}
		database.CreateUser(ctx, tx, user)

		site := &models.Site{
			Name:        "Original Site",
			Domain:      "original.com",
			CORSOrigins: []string{"https://original.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user.ID)

		// When: UpdateSite 호출
		requestBody := map[string]interface{}{
			"name":         "Updated Site",
			"cors_origins": []string{"https://updated.com"},
			"is_active":    false,
		}
		bodyBytes, _ := json.Marshal(requestBody)

		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodPut, "/admin/sites/"+strconv.FormatInt(site.ID, 10), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(ctx, userContextKey, user))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()

		handler.UpdateSite(rec, req)

		// Then: 200 OK
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["name"] != "Updated Site" {
			t.Errorf("Expected name 'Updated Site', got %v", response["name"])
		}
	})

	t.Run("사이트 수정 실패 - 소유자 아님", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 두 사용자와 사이트
		user1 := &models.User{
			Email:    "owner@example.com",
			Name:     "Owner",
			GoogleID: "google-owner-update",
		}
		database.CreateUser(ctx, tx, user1)

		user2 := &models.User{
			Email:    "notowner@example.com",
			Name:     "Not Owner",
			GoogleID: "google-notowner",
		}
		database.CreateUser(ctx, tx, user2)

		site := &models.Site{
			Name:        "Owner Site",
			Domain:      "ownersite.com",
			CORSOrigins: []string{"https://ownersite.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user1.ID)

		// When: user2가 수정 시도
		requestBody := map[string]interface{}{
			"name": "Hacked Site",
		}
		bodyBytes, _ := json.Marshal(requestBody)

		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodPut, "/admin/sites/"+strconv.FormatInt(site.ID, 10), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(ctx, userContextKey, user2))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()

		handler.UpdateSite(rec, req)

		// Then: 403 Forbidden
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

// TestDeleteSite는 사이트 삭제 기능을 테스트합니다
func TestDeleteSite(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("사이트 삭제 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자와 사이트 생성
		user := &models.User{
			Email:    "deleter@example.com",
			Name:     "Deleter",
			GoogleID: "google-deleter",
		}
		database.CreateUser(ctx, tx, user)

		site := &models.Site{
			Name:        "To Delete",
			Domain:      "todelete.com",
			CORSOrigins: []string{"https://todelete.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user.ID)

		// When: DeleteSite 호출
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodDelete, "/admin/sites/"+strconv.FormatInt(site.ID, 10), nil)
		req = req.WithContext(context.WithValue(ctx, userContextKey, user))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()

		handler.DeleteSite(rec, req)

		// Then: 204 No Content
		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
	})

	t.Run("사이트 삭제 실패 - 소유자 아님", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 두 사용자와 사이트
		user1 := &models.User{
			Email:    "owner-del@example.com",
			Name:     "Owner Del",
			GoogleID: "google-owner-del",
		}
		database.CreateUser(ctx, tx, user1)

		user2 := &models.User{
			Email:    "notowner-del@example.com",
			Name:     "Not Owner Del",
			GoogleID: "google-notowner-del",
		}
		database.CreateUser(ctx, tx, user2)

		site := &models.Site{
			Name:        "Owner Site Del",
			Domain:      "ownersitedel.com",
			CORSOrigins: []string{"https://ownersitedel.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user1.ID)

		// When: user2가 삭제 시도
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodDelete, "/admin/sites/"+strconv.FormatInt(site.ID, 10), nil)
		req = req.WithContext(context.WithValue(ctx, userContextKey, user2))

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()

		handler.DeleteSite(rec, req)

		// Then: 403 Forbidden
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

// TestGetProfile는 프로필 조회 기능을 테스트합니다
func TestGetProfile(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("프로필 조회 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자 생성
		user := &models.User{
			Email:      "profile@example.com",
			Name:       "Profile User",
			GoogleID:   "google-profile",
			PictureURL: "https://example.com/picture.jpg",
		}
		database.CreateUser(ctx, tx, user)

		// When: GetProfile 호출
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
		req = req.WithContext(context.WithValue(ctx, userContextKey, user))
		rec := httptest.NewRecorder()

		handler.GetProfile(rec, req)

		// Then: 200 OK
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["email"] != user.Email {
			t.Errorf("Expected email %s, got %v", user.Email, response["email"])
		}
		if response["name"] != user.Name {
			t.Errorf("Expected name %s, got %v", user.Name, response["name"])
		}
	})
}

// TestGetSiteStats는 사이트 통계 조회 기능을 테스트합니다
func TestGetSiteStats(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("통계 조회 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자 및 사이트 생성
		user := &models.User{
			Email:    "admin@example.com",
			Name:     "Admin",
			GoogleID: "google-admin",
		}
		if err := database.CreateUser(ctx, tx, user); err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		site := &models.Site{
			Name:        "Test Site",
			Domain:      "test.com",
			CORSOrigins: []string{"https://test.com"},
			IsActive:    true,
		}
		if err := database.CreateSiteForUser(ctx, tx, site, user.ID); err != nil {
			t.Fatalf("Failed to create site: %v", err)
		}

		// Post 생성 및 댓글 추가
		post := testhelpers.CreateTestPost(ctx, t, tx, site.ID, "test-post", "Test Post")
		for i := 0; i < 3; i++ {
			_, err := database.CreateComment(ctx, tx, post.ID, nil, "Author", "password123",
				"Comment", "127.0.0.1", "test-agent")
			if err != nil {
				t.Fatalf("Failed to create comment: %v", err)
			}
		}

		// When: GetSiteStats 호출
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/sites/"+strconv.FormatInt(site.ID, 10)+"/stats", nil)
		rec := httptest.NewRecorder()

		// Context에 user 추가
		ctx = context.WithValue(ctx, userContextKey, user)
		req = req.WithContext(ctx)

		// Chi URL 파라미터 추가
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetSiteStats(rec, req)

		// Then: 응답 검증
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var stats models.SiteStats
		json.Unmarshal(rec.Body.Bytes(), &stats)

		if stats.PostCount != 1 {
			t.Errorf("Expected post count 1, got %d", stats.PostCount)
		}
		if stats.CommentCount != 3 {
			t.Errorf("Expected comment count 3, got %d", stats.CommentCount)
		}
		if stats.DeletedCommentCount != 0 {
			t.Errorf("Expected deleted comment count 0, got %d", stats.DeletedCommentCount)
		}
	})

	t.Run("권한 없음 - 다른 사용자의 사이트", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 두 명의 사용자
		user1 := &models.User{
			Email:    "user1@example.com",
			Name:     "User 1",
			GoogleID: "google-user1",
		}
		user2 := &models.User{
			Email:    "user2@example.com",
			Name:     "User 2",
			GoogleID: "google-user2",
		}
		database.CreateUser(ctx, tx, user1)
		database.CreateUser(ctx, tx, user2)

		// user2의 사이트 생성
		site := &models.Site{
			Name:        "User2 Site",
			Domain:      "user2.com",
			CORSOrigins: []string{"https://user2.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user2.ID)

		// When: user1이 user2의 사이트 통계 조회 시도
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/sites/"+strconv.FormatInt(site.ID, 10)+"/stats", nil)
		rec := httptest.NewRecorder()

		// Context에 user1 추가
		ctx = context.WithValue(ctx, userContextKey, user1)
		req = req.WithContext(ctx)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetSiteStats(rec, req)

		// Then: 403 Forbidden
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})
}

func TestListSitePosts(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("포스트 목록 조회 성공", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자와 사이트, 여러 개의 포스트(댓글 포함)
		user := &models.User{
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-test",
		}
		database.CreateUser(ctx, tx, user)

		site := &models.Site{
			Name:        "Test Blog",
			Domain:      "test.com",
			CORSOrigins: []string{"https://test.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user.ID)

		post1 := testhelpers.CreateTestPost(ctx, t, tx, site.ID, "post-1", "Post 1")
		post2 := testhelpers.CreateTestPost(ctx, t, tx, site.ID, "post-2", "Post 2")
		_ = testhelpers.CreateTestPost(ctx, t, tx, site.ID, "post-3", "Post 3") // post3는 댓글 없음

		// post1: 활성 댓글 2개
		_, _ = database.CreateComment(ctx, tx, post1.ID, nil, "author1", "pass", "comment1", "1.1.1.1", "ua")
		_, _ = database.CreateComment(ctx, tx, post1.ID, nil, "author2", "pass", "comment2", "2.2.2.2", "ua")

		// post2: 활성 댓글 1개, 삭제된 댓글 1개
		_, _ = database.CreateComment(ctx, tx, post2.ID, nil, "author3", "pass", "comment3", "3.3.3.3", "ua")
		comment4, _ := database.CreateComment(ctx, tx, post2.ID, nil, "author4", "pass", "comment4", "4.4.4.4", "ua")
		_ = database.DeleteComment(ctx, tx, comment4.ID)

		// When: 포스트 목록 조회
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/sites/"+strconv.FormatInt(site.ID, 10)+"/posts", nil)
		rec := httptest.NewRecorder()

		ctx = context.WithValue(ctx, userContextKey, user)
		req = req.WithContext(ctx)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.ListSitePosts(rec, req)

		// Then: 200 OK 및 포스트 목록 반환
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", rec.Code)
		}

		var posts []*models.Post
		if err := json.NewDecoder(rec.Body).Decode(&posts); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(posts) != 3 {
			t.Errorf("Expected 3 posts, got %d", len(posts))
		}

		// post1 검증 (활성 댓글 2개)
		found := false
		for _, p := range posts {
			if p.ID == post1.ID {
				found = true
				if p.ActiveCommentCount != 2 {
					t.Errorf("post1: expected ActiveCommentCount 2, got %d", p.ActiveCommentCount)
				}
				if p.DeletedCommentCount != 0 {
					t.Errorf("post1: expected DeletedCommentCount 0, got %d", p.DeletedCommentCount)
				}
			}
		}
		if !found {
			t.Error("post1 not found in response")
		}

		// post2 검증 (활성 댓글 1개, 삭제된 댓글 1개)
		found = false
		for _, p := range posts {
			if p.ID == post2.ID {
				found = true
				if p.ActiveCommentCount != 1 {
					t.Errorf("post2: expected ActiveCommentCount 1, got %d", p.ActiveCommentCount)
				}
				if p.DeletedCommentCount != 1 {
					t.Errorf("post2: expected DeletedCommentCount 1, got %d", p.DeletedCommentCount)
				}
			}
		}
		if !found {
			t.Error("post2 not found in response")
		}
	})

	t.Run("권한 없음 - 다른 사용자의 사이트", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 두 명의 사용자
		user1 := &models.User{
			Email:    "user1@example.com",
			Name:     "User 1",
			GoogleID: "google-user1",
		}
		user2 := &models.User{
			Email:    "user2@example.com",
			Name:     "User 2",
			GoogleID: "google-user2",
		}
		database.CreateUser(ctx, tx, user1)
		database.CreateUser(ctx, tx, user2)

		// user2의 사이트 생성
		site := &models.Site{
			Name:        "User2 Blog",
			Domain:      "user2.com",
			CORSOrigins: []string{"https://user2.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user2.ID)

		// When: user1이 user2의 사이트 포스트 목록 조회 시도
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/sites/"+strconv.FormatInt(site.ID, 10)+"/posts", nil)
		rec := httptest.NewRecorder()

		ctx = context.WithValue(ctx, userContextKey, user1)
		req = req.WithContext(ctx)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", strconv.FormatInt(site.ID, 10))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.ListSitePosts(rec, req)

		// Then: 403 Forbidden
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})
}

func TestGetPostComments(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("포스트 댓글 조회 성공 - 삭제된 댓글 포함, 전체 IP", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자와 사이트, 포스트, 댓글(삭제된 것 포함)
		user := &models.User{
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-test",
		}
		database.CreateUser(ctx, tx, user)

		site := &models.Site{
			Name:        "Test Blog",
			Domain:      "test.com",
			CORSOrigins: []string{"https://test.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user.ID)

		post := testhelpers.CreateTestPost(ctx, t, tx, site.ID, "test-post", "Test Post")

		// 활성 댓글 2개
		comment1, _ := database.CreateComment(ctx, tx, post.ID, nil, "author1", "pass", "comment1", "1.1.1.1", "ua")
		_, _ = database.CreateComment(ctx, tx, post.ID, nil, "author2", "pass", "comment2", "2.2.2.2", "ua")

		// 삭제된 댓글 1개
		comment3, _ := database.CreateComment(ctx, tx, post.ID, nil, "author3", "pass", "deleted comment", "3.3.3.3", "ua")
		_ = database.DeleteComment(ctx, tx, comment3.ID)

		// 대댓글 1개 (활성)
		_, _ = database.CreateComment(ctx, tx, post.ID, &comment1.ID, "reply-author", "pass", "reply", "4.4.4.4", "ua")

		// When: 댓글 조회
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/posts/test-post/comments?site_id="+strconv.FormatInt(site.ID, 10), nil)
		rec := httptest.NewRecorder()

		ctx = context.WithValue(ctx, userContextKey, user)
		req = req.WithContext(ctx)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("slug", "test-post")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetPostComments(rec, req)

		// Then: 200 OK 및 댓글 목록 반환 (삭제된 것 포함)
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var response struct {
			Comments []*models.Comment `json:"comments"`
			Total    int               `json:"total"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Total != 3 {
			t.Errorf("Expected total 3, got %d", response.Total)
		}

		if len(response.Comments) != 3 {
			t.Errorf("Expected 3 comments, got %d", len(response.Comments))
		}

		// 삭제된 댓글이 포함되어 있는지 확인
		foundDeleted := false
		for _, c := range response.Comments {
			if c.ID == comment3.ID {
				foundDeleted = true
				if !c.IsDeleted {
					t.Error("Expected deleted comment to have IsDeleted=true")
				}
			}
		}
		if !foundDeleted {
			t.Error("Deleted comment not found in response")
		}

		// IP 주소가 마스킹되지 않았는지 확인 (admin은 전체 IP 확인 가능)
		// IPAddressUnmasked 필드에 전체 IP가 들어있어야 함
		for _, c := range response.Comments {
			if c.IPAddressUnmasked == "" || len(c.IPAddressUnmasked) < 7 { // "x.x.x.x" 형태가 아님
				t.Errorf("Expected full IP address for comment %d, got: %s", c.ID, c.IPAddressUnmasked)
			}
			// IPAddressMasked도 있어야 함
			if c.IPAddressMasked == "" {
				t.Errorf("Expected masked IP address for comment %d", c.ID)
			}
			// 대댓글도 IP 확인
			for _, reply := range c.Replies {
				if reply.IPAddressUnmasked == "" || len(reply.IPAddressUnmasked) < 7 {
					t.Errorf("Expected full IP address for reply %d, got: %s", reply.ID, reply.IPAddressUnmasked)
				}
				if reply.IPAddressMasked == "" {
					t.Errorf("Expected masked IP address for reply %d", reply.ID)
				}
			}
		}
	})

	t.Run("삭제된 대댓글도 작성자와 내용 원본 유지", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 사용자와 사이트, 포스트, 살아 있는 댓글 아래의 삭제된 대댓글
		user := &models.User{
			Email:    "test@example.com",
			Name:     "Test User",
			GoogleID: "google-test",
		}
		database.CreateUser(ctx, tx, user)

		site := &models.Site{
			Name:        "Test Blog",
			Domain:      "test.com",
			CORSOrigins: []string{"https://test.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user.ID)

		post := testhelpers.CreateTestPost(ctx, t, tx, site.ID, "test-post", "Test Post")

		parent, _ := database.CreateComment(ctx, tx, post.ID, nil, "parent-author", "pass", "parent", "1.1.1.1", "ua")
		reply, _ := database.CreateComment(ctx, tx, post.ID, &parent.ID, "reply-author", "pass", "deleted reply", "2.2.2.2", "ua")
		_ = database.DeleteComment(ctx, tx, reply.ID)

		// When: 댓글 조회
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/posts/test-post/comments?site_id="+strconv.FormatInt(site.ID, 10), nil)
		rec := httptest.NewRecorder()

		ctx = context.WithValue(ctx, userContextKey, user)
		req = req.WithContext(ctx)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("slug", "test-post")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetPostComments(rec, req)

		// Then: 삭제된 대댓글의 작성자와 내용이 원본 그대로
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var response struct {
			Comments []*models.Comment `json:"comments"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Comments) != 1 || len(response.Comments[0].Replies) != 1 {
			t.Fatalf("Expected 1 comment with 1 reply, got %+v", response.Comments)
		}

		deletedReply := response.Comments[0].Replies[0]
		if !deletedReply.IsDeleted {
			t.Error("Expected reply IsDeleted=true")
		}
		if deletedReply.AuthorName != "reply-author" {
			t.Errorf("Expected original author_name 'reply-author', got '%s'", deletedReply.AuthorName)
		}
		if deletedReply.Content != "deleted reply" {
			t.Errorf("Expected original content 'deleted reply', got '%s'", deletedReply.Content)
		}
	})

	t.Run("권한 없음 - 다른 사용자의 사이트", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 두 명의 사용자
		user1 := &models.User{
			Email:    "user1@example.com",
			Name:     "User 1",
			GoogleID: "google-user1",
		}
		user2 := &models.User{
			Email:    "user2@example.com",
			Name:     "User 2",
			GoogleID: "google-user2",
		}
		database.CreateUser(ctx, tx, user1)
		database.CreateUser(ctx, tx, user2)

		// user2의 사이트와 포스트 생성
		site := &models.Site{
			Name:        "User2 Blog",
			Domain:      "user2.com",
			CORSOrigins: []string{"https://user2.com"},
			IsActive:    true,
		}
		database.CreateSiteForUser(ctx, tx, site, user2.ID)

		_ = testhelpers.CreateTestPost(ctx, t, tx, site.ID, "user2-post", "User2 Post")

		// When: user1이 user2의 포스트 댓글 조회 시도
		handler := NewAdminHandler(tx)
		req := httptest.NewRequest(http.MethodGet, "/admin/posts/user2-post/comments?site_id="+strconv.FormatInt(site.ID, 10), nil)
		rec := httptest.NewRecorder()

		ctx = context.WithValue(ctx, userContextKey, user1)
		req = req.WithContext(ctx)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("slug", "user2-post")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetPostComments(rec, req)

		// Then: 403 Forbidden
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})
}

// adminDeleteFixture는 어드민 댓글 삭제 테스트용 사용자·사이트·포스트입니다
type adminDeleteFixture struct {
	owner *models.User
	site  *models.Site
	post  models.Post
}

// setupAdminDeleteFixture는 소유자와 소유자의 사이트, 포스트를 만듭니다
func setupAdminDeleteFixture(ctx context.Context, t *testing.T, tx database.DBTX) adminDeleteFixture {
	t.Helper()

	owner := &models.User{Email: "owner@example.com", Name: "Owner", GoogleID: "google-owner"}
	if err := database.CreateUser(ctx, tx, owner); err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}

	site := &models.Site{
		Name:        "Owner Blog",
		Domain:      "owner.com",
		CORSOrigins: []string{"https://owner.com"},
		IsActive:    true,
	}
	if err := database.CreateSiteForUser(ctx, tx, site, owner.ID); err != nil {
		t.Fatalf("Failed to create site: %v", err)
	}

	post := testhelpers.CreateTestPost(ctx, t, tx, site.ID, "post-1", "Post 1")
	return adminDeleteFixture{owner: owner, site: site, post: post}
}

// createStrangerWithSite는 자기 사이트를 가진 다른 사용자를 만듭니다 (사이트 간 격리 검증용)
func createStrangerWithSite(ctx context.Context, t *testing.T, tx database.DBTX) *models.User {
	t.Helper()

	stranger := &models.User{Email: "stranger@example.com", Name: "Stranger", GoogleID: "google-stranger"}
	if err := database.CreateUser(ctx, tx, stranger); err != nil {
		t.Fatalf("Failed to create stranger: %v", err)
	}

	strangerSite := &models.Site{
		Name:        "Stranger Blog",
		Domain:      "stranger.com",
		CORSOrigins: []string{"https://stranger.com"},
		IsActive:    true,
	}
	if err := database.CreateSiteForUser(ctx, tx, strangerSite, stranger.ID); err != nil {
		t.Fatalf("Failed to create stranger site: %v", err)
	}

	return stranger
}

// requestAdminDeleteComment는 user를 context에 넣고 DELETE /admin/comments/{id}를 호출합니다
// user가 nil이면 context에 사용자를 넣지 않습니다
func requestAdminDeleteComment(ctx context.Context, tx database.DBTX, user *models.User, commentID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, "/admin/comments/"+commentID, nil)
	if user != nil {
		ctx = context.WithValue(ctx, userContextKey, user)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", commentID)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	NewAdminHandler(tx).DeleteComment(rec, req)
	return rec
}

// TestAdminDeleteComment는 어드민 댓글 삭제 기능을 테스트합니다
func TestAdminDeleteComment(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)

	t.Run("삭제 성공 - 204, soft delete 기록", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자 사이트의 댓글
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "spammer", "pass", "spam", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		// When: 소유자가 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, strconv.FormatInt(comment.ID, 10))

		// Then: 204, is_deleted와 deleted_at 기록
		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
		}
		deleted, err := database.GetCommentByID(ctx, tx, comment.ID)
		if err != nil {
			t.Fatalf("Failed to get comment: %v", err)
		}
		if !deleted.IsDeleted {
			t.Error("Expected is_deleted=true, got false")
		}
		if deleted.DeletedAt == nil {
			t.Error("Expected deleted_at to be set, got nil")
		}
	})

	t.Run("이미 삭제된 댓글 - 204 멱등", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 이미 삭제된 댓글
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "author", "pass", "content", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		if err := database.DeleteComment(ctx, tx, comment.ID); err != nil {
			t.Fatalf("Failed to delete comment: %v", err)
		}

		// When: 다시 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, strconv.FormatInt(comment.ID, 10))

		// Then: 204
		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
		}
	})

	t.Run("없는 댓글 - 404", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자만 존재
		f := setupAdminDeleteFixture(ctx, t, tx)

		// When: 없는 ID 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, "999999999")

		// Then: 404
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNotFound, rec.Code, rec.Body.String())
		}
	})

	t.Run("다른 사용자 사이트의 댓글 - 403, 삭제되지 않음", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자 사이트의 댓글과, 자기 사이트를 가진 다른 사용자
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "author", "pass", "content", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		stranger := createStrangerWithSite(ctx, t, tx)

		// When: 다른 사용자가 삭제
		rec := requestAdminDeleteComment(ctx, tx, stranger, strconv.FormatInt(comment.ID, 10))

		// Then: 403, 댓글은 그대로
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusForbidden, rec.Code, rec.Body.String())
		}
		unchanged, err := database.GetCommentByID(ctx, tx, comment.ID)
		if err != nil {
			t.Fatalf("Failed to get comment: %v", err)
		}
		if unchanged.IsDeleted {
			t.Error("Expected comment to remain, but it was deleted")
		}
	})

	t.Run("다른 사용자 사이트의 삭제된 댓글 - 403", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자 사이트의 삭제된 댓글과, 자기 사이트를 가진 다른 사용자
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "author", "pass", "content", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		if err := database.DeleteComment(ctx, tx, comment.ID); err != nil {
			t.Fatalf("Failed to delete comment: %v", err)
		}
		stranger := createStrangerWithSite(ctx, t, tx)

		// When: 다른 사용자가 삭제
		rec := requestAdminDeleteComment(ctx, tx, stranger, strconv.FormatInt(comment.ID, 10))

		// Then: 삭제 여부를 드러내지 않고 403
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusForbidden, rec.Code, rec.Body.String())
		}
	})

	t.Run("잘못된 ID - 400", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자
		f := setupAdminDeleteFixture(ctx, t, tx)

		for _, invalidID := range []string{"abc", "0", "-1"} {
			// When: 잘못된 ID로 삭제
			rec := requestAdminDeleteComment(ctx, tx, f.owner, invalidID)

			// Then: 400
			if rec.Code != http.StatusBadRequest {
				t.Errorf("id=%q: Expected status %d, got %d", invalidID, http.StatusBadRequest, rec.Code)
			}
		}
	})

	t.Run("사용자 context 없음 - 401", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 소유자 사이트의 댓글
		f := setupAdminDeleteFixture(ctx, t, tx)
		comment, err := database.CreateComment(ctx, tx, f.post.ID, nil, "author", "pass", "content", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		// When: 사용자 없이 삭제
		rec := requestAdminDeleteComment(ctx, tx, nil, strconv.FormatInt(comment.ID, 10))

		// Then: 401
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("대댓글이 달린 부모 삭제 - 대댓글 유지", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 부모 댓글과 대댓글
		f := setupAdminDeleteFixture(ctx, t, tx)
		parent, err := database.CreateComment(ctx, tx, f.post.ID, nil, "parent", "pass", "parent", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create parent: %v", err)
		}
		reply, err := database.CreateComment(ctx, tx, f.post.ID, &parent.ID, "reply", "pass", "reply", "2.2.2.2", "ua")
		if err != nil {
			t.Fatalf("Failed to create reply: %v", err)
		}

		// When: 부모 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, strconv.FormatInt(parent.ID, 10))

		// Then: 204, 대댓글은 삭제되지 않음
		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
		}
		remaining, err := database.GetCommentByID(ctx, tx, reply.ID)
		if err != nil {
			t.Fatalf("Failed to get reply: %v", err)
		}
		if remaining.IsDeleted {
			t.Error("Expected reply to remain, but it was deleted")
		}
	})

	t.Run("삭제 후 사이트 통계에 반영", func(t *testing.T) {
		ctx, tx, cleanup := testhelpers.SetupTxTest(t, db)
		defer cleanup()

		// Given: 활성 댓글 2개
		f := setupAdminDeleteFixture(ctx, t, tx)
		target, err := database.CreateComment(ctx, tx, f.post.ID, nil, "a", "pass", "a", "1.1.1.1", "ua")
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		if _, err := database.CreateComment(ctx, tx, f.post.ID, nil, "b", "pass", "b", "2.2.2.2", "ua"); err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		// When: 한 개 삭제
		rec := requestAdminDeleteComment(ctx, tx, f.owner, strconv.FormatInt(target.ID, 10))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusNoContent, rec.Code, rec.Body.String())
		}

		// Then: 활성 1, 삭제 1
		stats, err := database.GetSiteStats(ctx, tx, f.site.ID)
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}
		if stats.CommentCount != 1 || stats.DeletedCommentCount != 1 {
			t.Errorf("Expected comment_count=1, deleted_comment_count=1, got %d, %d", stats.CommentCount, stats.DeletedCommentCount)
		}
	})
}
