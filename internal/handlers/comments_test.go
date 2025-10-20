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
// ListComments 테스트
// ============================================

func TestListComments_Success_TreeStructure(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 사이트, 포스트, 댓글 계층 구조 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "list.test.com", []string{"http://localhost:3000"}, true)
	site, _ := database.GetSiteByAPIKey(db, apiKey)
	post, _ := database.GetOrCreatePost(db, site.ID, "test-post", "Test Post")

	// 최상위 댓글 2개
	parent1, _ := database.CreateComment(db, post.ID, nil, "Parent1", "pass123", "첫 번째 댓글", "127.0.0.1", "Agent")
	_, _ = database.CreateComment(db, post.ID, nil, "Parent2", "pass123", "두 번째 댓글", "127.0.0.1", "Agent")

	// parent1에 대댓글 2개
	database.CreateComment(db, post.ID, &parent1.ID, "Child1", "pass123", "첫 번째 대댓글", "127.0.0.1", "Agent")
	database.CreateComment(db, post.ID, &parent1.ID, "Child2", "pass123", "두 번째 대댓글", "127.0.0.1", "Agent")

	handler := NewCommentHandler(db)

	// HTTP 요청 생성
	req := httptest.NewRequest(http.MethodGet, "/api/posts/test-post/comments?page=1&limit=50", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	// Chi URL 파라미터 설정
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: ListComments 호출
	handler.ListComments(rec, req)

	// Then: 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// 응답 파싱
	var response struct {
		Comments []struct {
			ID         int64  `json:"id"`
			AuthorName string `json:"author_name"`
			Content    string `json:"content"`
			Replies    []struct {
				ID         int64  `json:"id"`
				AuthorName string `json:"author_name"`
				Content    string `json:"content"`
				ParentID   *int64 `json:"parent_id"`
			} `json:"replies"`
		} `json:"comments"`
		Pagination struct {
			CurrentPage   int `json:"current_page"`
			TotalPages    int `json:"total_pages"`
			TotalComments int `json:"total_comments"`
			PerPage       int `json:"per_page"`
		} `json:"pagination"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// 최상위 댓글 2개
	if len(response.Comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(response.Comments))
	}

	// 첫 번째 댓글에 대댓글 2개
	if len(response.Comments[0].Replies) != 2 {
		t.Errorf("Expected 2 replies for first comment, got %d", len(response.Comments[0].Replies))
	}

	// 두 번째 댓글에 대댓글 0개
	if len(response.Comments[1].Replies) != 0 {
		t.Errorf("Expected 0 replies for second comment, got %d", len(response.Comments[1].Replies))
	}

	// parent_id 확인
	if *response.Comments[0].Replies[0].ParentID != parent1.ID {
		t.Errorf("Expected parent_id %d, got %d", parent1.ID, *response.Comments[0].Replies[0].ParentID)
	}

	// 페이지네이션 확인
	if response.Pagination.TotalComments != 2 {
		t.Errorf("Expected total_comments 2, got %d", response.Pagination.TotalComments)
	}
	if response.Pagination.CurrentPage != 1 {
		t.Errorf("Expected current_page 1, got %d", response.Pagination.CurrentPage)
	}
}

func TestListComments_Success_Pagination(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 사이트, 포스트, 최상위 댓글 3개 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "pagination.test.com", []string{"http://localhost:3000"}, true)
	site, _ := database.GetSiteByAPIKey(db, apiKey)
	post, _ := database.GetOrCreatePost(db, site.ID, "test-post", "Test Post")

	for i := 1; i <= 3; i++ {
		database.CreateComment(db, post.ID, nil, "Author", "pass123", "댓글 내용", "127.0.0.1", "Agent")
	}

	handler := NewCommentHandler(db)

	// limit=2로 첫 페이지 조회
	req := httptest.NewRequest(http.MethodGet, "/api/posts/test-post/comments?page=1&limit=2", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: ListComments 호출
	handler.ListComments(rec, req)

	// Then: 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response struct {
		Comments   []interface{} `json:"comments"`
		Pagination struct {
			CurrentPage   int `json:"current_page"`
			TotalPages    int `json:"total_pages"`
			TotalComments int `json:"total_comments"`
			PerPage       int `json:"per_page"`
		} `json:"pagination"`
	}

	json.NewDecoder(rec.Body).Decode(&response)

	// 페이지네이션 확인
	if len(response.Comments) != 2 {
		t.Errorf("Expected 2 comments in first page, got %d", len(response.Comments))
	}
	if response.Pagination.TotalComments != 3 {
		t.Errorf("Expected total_comments 3, got %d", response.Pagination.TotalComments)
	}
	if response.Pagination.TotalPages != 2 {
		t.Errorf("Expected total_pages 2, got %d", response.Pagination.TotalPages)
	}
	if response.Pagination.PerPage != 2 {
		t.Errorf("Expected per_page 2, got %d", response.Pagination.PerPage)
	}
}

func TestListComments_Success_EmptyPost(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 사이트만 생성 (포스트 없음)
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "empty.test.com", []string{"http://localhost:3000"}, true)
	site, _ := database.GetSiteByAPIKey(db, apiKey)

	handler := NewCommentHandler(db)

	// 존재하지 않는 포스트 조회
	req := httptest.NewRequest(http.MethodGet, "/api/posts/nonexistent/comments", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: ListComments 호출
	handler.ListComments(rec, req)

	// Then: 200 OK with empty array
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response struct {
		Comments   []interface{} `json:"comments"`
		Pagination struct {
			TotalComments int `json:"total_comments"`
		} `json:"pagination"`
	}

	json.NewDecoder(rec.Body).Decode(&response)

	// 빈 배열
	if len(response.Comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(response.Comments))
	}
	if response.Pagination.TotalComments != 0 {
		t.Errorf("Expected total_comments 0, got %d", response.Pagination.TotalComments)
	}
}

func TestListComments_DeletedComments(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	defer database.Close(db)
	defer testhelpers.CleanupSites(t, db)

	// Given: 사이트, 포스트, 댓글 생성
	apiKey := testhelpers.CreateTestSite(t, db, "Test Site", "deleted.test.com", []string{"http://localhost:3000"}, true)
	site, _ := database.GetSiteByAPIKey(db, apiKey)
	post, _ := database.GetOrCreatePost(db, site.ID, "test-post", "Test Post")

	// 최상위 댓글 (대댓글 있음)
	parent, _ := database.CreateComment(db, post.ID, nil, "Parent", "pass123", "부모 댓글", "127.0.0.1", "Agent")
	database.CreateComment(db, post.ID, &parent.ID, "Child", "pass123", "자식 댓글", "127.0.0.1", "Agent")

	// 최상위 댓글 (대댓글 없음)
	alone, _ := database.CreateComment(db, post.ID, nil, "Alone", "pass123", "혼자 댓글", "127.0.0.1", "Agent")

	// 삭제 처리
	database.DeleteComment(db, parent.ID)
	database.DeleteComment(db, alone.ID)

	handler := NewCommentHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/posts/test-post/comments", nil)
	req.Header.Set("X-Orbithall-API-Key", apiKey)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", "test-post")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(withSiteContext(req.Context(), site))

	rec := httptest.NewRecorder()

	// When: ListComments 호출
	handler.ListComments(rec, req)

	// Then: 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response struct {
		Comments []struct {
			ID         int64  `json:"id"`
			AuthorName string `json:"author_name"`
			Content    string `json:"content"`
			IsDeleted  bool   `json:"is_deleted"`
			Replies    []interface{} `json:"replies"`
		} `json:"comments"`
	}

	json.NewDecoder(rec.Body).Decode(&response)

	// 대댓글이 있는 parent는 빈 값으로 포함 (isDeleted=true)
	if len(response.Comments) != 1 {
		t.Errorf("Expected 1 comment (deleted with replies), got %d", len(response.Comments))
	}

	// 삭제된 댓글은 author_name과 content가 빈 문자열
	if response.Comments[0].AuthorName != "" {
		t.Errorf("Expected author_name empty string, got '%s'", response.Comments[0].AuthorName)
	}

	if response.Comments[0].Content != "" {
		t.Errorf("Expected content empty string, got '%s'", response.Comments[0].Content)
	}

	// isDeleted 플래그로 삭제된 댓글임을 클라이언트가 판단
	if !response.Comments[0].IsDeleted {
		t.Error("Expected is_deleted true")
	}

	// alone은 대댓글이 없으므로 목록에서 완전히 제외됨
}

// ============================================
// 테스트 헬퍼
// ============================================

// withSiteContext는 테스트용으로 Context에 사이트 정보를 추가합니다
// middleware.go의 AuthMiddleware가 하는 것과 동일한 방식
func withSiteContext(ctx context.Context, site *models.Site) context.Context {
	return context.WithValue(ctx, siteContextKey, site)
}
