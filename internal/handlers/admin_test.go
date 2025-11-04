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
