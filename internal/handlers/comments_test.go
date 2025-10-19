package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/testhelpers"
)

// ============================================
// CreateComment 테스트
// ============================================

func TestCreateComment_Success_TopLevel(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 활성 사이트 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "toplevel.test.com", []string{"http://localhost:3000"}, true)

	// 핸들러 생성
	handler := NewCommentHandler(db)

	// 테스트 요청 데이터
	requestBody := map[string]interface{}{
		"author_name": "홍길동",
		"password":    "test1234",
		"content":     "좋은 글 잘 읽었습니다!",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	// HTTP 요청 생성
	req := httptest.NewRequest(http.MethodPost, "/api/posts/test-post/comments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	// Chi URL 파라미터 설정
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// 미들웨어로 사이트 정보 주입
	site, _ := database.GetSiteByAPIKey(db, apiKey)
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: CreateComment 호출
	handler.CreateComment(rec, req)

	// Then: 201 Created
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	// 응답 파싱
	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// 필드 검증
	if response["author_name"] != "홍길동" {
		t.Errorf("Expected author_name '홍길동', got %v", response["author_name"])
	}
	if response["content"] != "좋은 글 잘 읽었습니다!" {
		t.Errorf("Expected content '좋은 글 잘 읽었습니다!', got %v", response["content"])
	}
	if response["parent_id"] != nil {
		t.Errorf("Expected parent_id nil, got %v", response["parent_id"])
	}
	if response["is_deleted"] != false {
		t.Errorf("Expected is_deleted false, got %v", response["is_deleted"])
	}

	// ID가 생성되었는지 확인
	if response["id"] == nil || response["id"].(float64) == 0 {
		t.Error("Expected non-zero id")
	}

	// IP 마스킹 확인
	if ipMasked, ok := response["ip_address_masked"].(string); ok {
		if ipMasked == "" {
			t.Error("Expected ip_address_masked to be set")
		}
	} else {
		t.Error("Expected ip_address_masked field")
	}
}

func TestCreateComment_Success_Reply(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 사이트 및 부모 댓글 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "reply.test.com", []string{"http://localhost:3000"}, true)
	site, _ := database.GetSiteByAPIKey(db, apiKey)

	// 부모 댓글 생성 (포스트 자동 생성됨)
	post, _ := database.GetOrCreatePost(db, site.ID, "test-post", "Test Post")
	parentComment, _ := database.CreateComment(db, post.ID, nil, "Parent Author", "password123", "Parent comment", "127.0.0.1", "Test Agent")

	handler := NewCommentHandler(db)

	// 대댓글 요청
	requestBody := map[string]interface{}{
		"author_name": "자식 작성자",
		"password":    "reply1234",
		"content":     "부모 댓글에 대한 답변",
		"parent_id":   parentComment.ID,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/posts/test-post/comments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: CreateComment 호출
	handler.CreateComment(rec, req)

	// Then: 201 Created
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	// parent_id 확인
	if response["parent_id"] == nil {
		t.Error("Expected parent_id to be set")
	} else if int64(response["parent_id"].(float64)) != parentComment.ID {
		t.Errorf("Expected parent_id %d, got %v", parentComment.ID, response["parent_id"])
	}
}

func TestCreateComment_Fail_2DepthReply(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 사이트, 부모 댓글, 자식 댓글 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "2depth.test.com", []string{"http://localhost:3000"}, true)
	site, _ := database.GetSiteByAPIKey(db, apiKey)

	post, _ := database.GetOrCreatePost(db, site.ID, "test-post", "Test Post")
	parentComment, _ := database.CreateComment(db, post.ID, nil, "Parent", "pass123", "Parent", "127.0.0.1", "Agent")
	childComment, _ := database.CreateComment(db, post.ID, &parentComment.ID, "Child", "pass123", "Child", "127.0.0.1", "Agent")

	handler := NewCommentHandler(db)

	// 손자 댓글 시도 (2-depth, 금지됨)
	requestBody := map[string]interface{}{
		"author_name": "손자",
		"password":    "pass123",
		"content":     "손자 댓글",
		"parent_id":   childComment.ID,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/posts/test-post/comments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: CreateComment 호출
	handler.CreateComment(rec, req)

	// Then: 400 Bad Request
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Error.Code != ErrInvalidInput {
		t.Errorf("Expected error code %s, got %s", ErrInvalidInput, response.Error.Code)
	}
}

func TestCreateComment_Fail_ValidationError(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 사이트 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "validation.test.com", []string{"http://localhost:3000"}, true)
	site, _ := database.GetSiteByAPIKey(db, apiKey)

	handler := NewCommentHandler(db)

	// 잘못된 요청 (author_name 없음, password 짧음, content 없음)
	requestBody := map[string]interface{}{
		"author_name": "",
		"password":    "123",
		"content":     "",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/posts/test-post/comments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: CreateComment 호출
	handler.CreateComment(rec, req)

	// Then: 400 Bad Request
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)

	if response.Error.Code != ErrInvalidInput {
		t.Errorf("Expected error code %s, got %s", ErrInvalidInput, response.Error.Code)
	}

	// details에 여러 필드 에러가 있어야 함
	if response.Error.Details == nil {
		t.Error("Expected validation error details")
	}
}

func TestCreateComment_XSS_HTMLSanitization(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 사이트 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "xss.test.com", []string{"http://localhost:3000"}, true)
	site, _ := database.GetSiteByAPIKey(db, apiKey)

	handler := NewCommentHandler(db)

	// XSS 공격 시도
	requestBody := map[string]interface{}{
		"author_name": "Hacker",
		"password":    "hack1234",
		"content":     "<script>alert('XSS')</script>안전한 내용",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/posts/test-post/comments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: CreateComment 호출
	handler.CreateComment(rec, req)

	// Then: 201 Created
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	// HTML 태그가 제거되어야 함
	content := response["content"].(string)
	if content != "안전한 내용" {
		t.Errorf("Expected sanitized content '안전한 내용', got '%s'", content)
	}
}

func TestCreateComment_Fail_ParentNotFound(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 사이트 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "parent.test.com", []string{"http://localhost:3000"}, true)
	site, _ := database.GetSiteByAPIKey(db, apiKey)

	handler := NewCommentHandler(db)

	// 존재하지 않는 부모 댓글 ID
	requestBody := map[string]interface{}{
		"author_name": "작성자",
		"password":    "pass1234",
		"content":     "댓글 내용",
		"parent_id":   99999,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/posts/test-post/comments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: CreateComment 호출
	handler.CreateComment(rec, req)

	// Then: 404 Not Found 또는 400 Bad Request
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 404 or 400, got %d", rec.Code)
	}
}

// ============================================
// 테스트 헬퍼
// ============================================

// withSiteContext는 테스트용으로 Context에 사이트 정보를 추가합니다
// middleware.go의 AuthMiddleware가 하는 것과 동일한 방식
func withSiteContext(ctx context.Context, site *models.Site) context.Context {
	return context.WithValue(ctx, siteContextKey, site)
}
