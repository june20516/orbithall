package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/validators"
)

// AdminHandler는 Admin API 요청을 처리합니다
type AdminHandler struct {
	db database.DBTX
}

// NewAdminHandler는 AdminHandler의 새 인스턴스를 생성합니다
func NewAdminHandler(db database.DBTX) *AdminHandler {
	return &AdminHandler{
		db: db,
	}
}

// ListSitesResponse는 사이트 목록 응답입니다
type ListSitesResponse struct {
	Sites []models.Site `json:"sites"`
}

// ListSites는 JWT 인증된 사용자의 사이트 목록을 반환합니다
// @Summary      내 사이트 목록 조회
// @Description  JWT 인증된 사용자가 소유한 모든 사이트 목록을 반환합니다
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200 {object} ListSitesResponse
// @Failure      401 {string} string "Unauthorized"
// @Failure      500 {string} string "Failed to get sites"
// @Security     BearerAuth
// @Router       /admin/sites [get]
func (h *AdminHandler) ListSites(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 사용자의 사이트 목록 조회
	sites, err := database.GetUserSites(r.Context(), h.db, user.ID)
	if err != nil {
		http.Error(w, "Failed to get sites", http.StatusInternalServerError)
		return
	}

	// 응답 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ListSitesResponse{Sites: sites})
}

// GetSite는 특정 사이트 상세 정보를 반환합니다
// @Summary      사이트 상세 조회
// @Description  특정 사이트의 상세 정보를 반환합니다 (소유자만 접근 가능)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id  path     int  true  "Site ID"
// @Success      200 {object} models.Site
// @Failure      400 {string} string "Invalid site ID"
// @Failure      401 {string} string "Unauthorized"
// @Failure      404 {string} string "Site not found"
// @Failure      500 {string} string "Failed to get site"
// @Security     BearerAuth
// @Router       /admin/sites/{id} [get]
func (h *AdminHandler) GetSite(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// URL 파라미터에서 site_id 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	// 사이트 조회
	site, err := database.GetSiteByID(r.Context(), h.db, siteID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Site not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get site", http.StatusInternalServerError)
		return
	}

	// 접근 권한 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		http.Error(w, "Failed to check access", http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	// 응답 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(site)
}

// CreateSite는 새 사이트를 생성합니다
// @Summary      사이트 생성
// @Description  새로운 사이트를 생성하고 API Key를 발급합니다
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        site body validators.SiteCreateInput true "사이트 생성 정보"
// @Success      201 {object} models.Site
// @Failure      400 {object} map[string]interface{} "Invalid input"
// @Failure      401 {string} string "Unauthorized"
// @Failure      500 {string} string "Failed to create site"
// @Security     BearerAuth
// @Router       /admin/sites [post]
func (h *AdminHandler) CreateSite(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Content-Type 검증
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// JSON 요청 파싱
	var input validators.SiteCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 입력 검증
	if err := input.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 사이트 생성
	site := &models.Site{
		Name:        input.Name,
		Domain:      input.Domain,
		CORSOrigins: input.CORSOrigins,
		IsActive:    true,
	}

	err := database.CreateSiteForUser(r.Context(), h.db, site, user.ID)
	if err != nil {
		http.Error(w, "Failed to create site", http.StatusInternalServerError)
		return
	}

	// 201 Created 응답
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(site)
}

// UpdateSite는 사이트 정보를 수정합니다
// @Summary      사이트 수정
// @Description  사이트 정보를 수정합니다 (소유자만 접근 가능, domain과 api_key는 수정 불가)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id   path     int  true  "Site ID"
// @Param        site body validators.SiteUpdateInput true "사이트 수정 정보"
// @Success      200 {object} models.Site
// @Failure      400 {object} map[string]interface{} "Invalid input"
// @Failure      401 {string} string "Unauthorized"
// @Failure      403 {string} string "Forbidden"
// @Failure      404 {string} string "Site not found"
// @Failure      500 {string} string "Failed to update site"
// @Security     BearerAuth
// @Router       /admin/sites/{id} [put]
func (h *AdminHandler) UpdateSite(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// URL 파라미터에서 site_id 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	// Content-Type 검증
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// JSON 요청 파싱
	var input validators.SiteUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 입력 검증
	if err := input.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// 접근 권한 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		http.Error(w, "Failed to check access", http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 기존 사이트 조회 (현재 값 가져오기)
	site, err := database.GetSiteByID(r.Context(), h.db, siteID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Site not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get site", http.StatusInternalServerError)
		return
	}

	// 수정할 필드 결정 (제공된 필드만 수정)
	name := site.Name
	corsOrigins := site.CORSOrigins
	isActive := site.IsActive

	if input.Name != nil {
		name = *input.Name
	}
	if input.CORSOrigins != nil {
		corsOrigins = *input.CORSOrigins
	}
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	// 사이트 수정
	err = database.UpdateSite(r.Context(), h.db, siteID, name, corsOrigins, isActive)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Site not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update site", http.StatusInternalServerError)
		return
	}

	// 수정된 사이트 재조회
	updatedSite, err := database.GetSiteByID(r.Context(), h.db, siteID)
	if err != nil {
		http.Error(w, "Failed to get updated site", http.StatusInternalServerError)
		return
	}

	// 200 OK 응답
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedSite)
}

// DeleteSite는 사이트를 삭제합니다
// @Summary      사이트 삭제
// @Description  사이트를 삭제합니다 (CASCADE로 연결된 posts, comments도 삭제됨)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Site ID"
// @Success      204 "No Content"
// @Failure      400 {string} string "Invalid site ID"
// @Failure      401 {string} string "Unauthorized"
// @Failure      403 {string} string "Forbidden"
// @Failure      404 {string} string "Site not found"
// @Failure      500 {string} string "Failed to delete site"
// @Security     BearerAuth
// @Router       /admin/sites/{id} [delete]
func (h *AdminHandler) DeleteSite(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// URL 파라미터에서 site_id 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	// 접근 권한 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		http.Error(w, "Failed to check access", http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 사이트 삭제
	err = database.DeleteSite(r.Context(), h.db, siteID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Site not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete site", http.StatusInternalServerError)
		return
	}

	// 204 No Content 응답
	w.WriteHeader(http.StatusNoContent)
}

// GetProfile는 현재 인증된 사용자의 프로필을 반환합니다
// @Summary      내 프로필 조회
// @Description  현재 JWT 인증된 사용자의 프로필 정보를 반환합니다
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200 {object} models.User
// @Failure      401 {string} string "Unauthorized"
// @Security     BearerAuth
// @Router       /admin/profile [get]
func (h *AdminHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 응답 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// GetSiteStats는 특정 사이트의 통계를 반환합니다
// @Summary      사이트 통계 조회
// @Description  특정 사이트의 포스트 수, 댓글 수, 삭제된 댓글 수를 반환합니다
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Site ID"
// @Success      200 {object} models.SiteStats
// @Failure      400 {string} string "Invalid site ID"
// @Failure      403 {string} string "Forbidden"
// @Failure      500 {string} string "Failed to get site stats"
// @Security     BearerAuth
// @Router       /admin/sites/{id}/stats [get]
func (h *AdminHandler) GetSiteStats(w http.ResponseWriter, r *http.Request) {
	// URL에서 site ID 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 사용자가 해당 사이트에 접근 권한이 있는지 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		http.Error(w, "Failed to check site access", http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 통계 조회
	stats, err := database.GetSiteStats(r.Context(), h.db, siteID)
	if err != nil {
		http.Error(w, "Failed to get site stats", http.StatusInternalServerError)
		return
	}

	// 응답 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// ListSitePosts는 특정 사이트의 포스트 목록을 반환합니다
// @Summary      사이트 포스트 목록 조회
// @Description  특정 사이트의 포스트 목록을 활성/삭제 댓글 수와 함께 반환합니다
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Site ID"
// @Success      200 {array} models.Post
// @Failure      400 {string} string "Invalid site ID"
// @Failure      403 {string} string "Forbidden"
// @Failure      500 {string} string "Failed to get posts"
// @Security     BearerAuth
// @Router       /admin/sites/{id}/posts [get]
func (h *AdminHandler) ListSitePosts(w http.ResponseWriter, r *http.Request) {
	// URL에서 site ID 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 사용자가 해당 사이트에 접근 권한이 있는지 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		http.Error(w, "Failed to check site access", http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 포스트 목록 조회
	posts, err := database.ListPostsBySite(r.Context(), h.db, siteID)
	if err != nil {
		http.Error(w, "Failed to get posts", http.StatusInternalServerError)
		return
	}

	// 응답 반환
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(posts)
}

// GetPostComments는 특정 포스트의 댓글 목록을 반환합니다 (Admin용 - 삭제된 댓글 포함, 전체 IP)
// @Summary      포스트 댓글 목록 조회
// @Description  특정 포스트의 모든 댓글을 조회합니다. 삭제된 댓글도 포함하고 IP 주소는 마스킹하지 않습니다.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        slug path string true "Post Slug"
// @Param        site_id query int true "Site ID"
// @Param        limit query int false "댓글 개수 (기본값: 50)"
// @Param        offset query int false "오프셋 (기본값: 0)"
// @Success      200 {object} object{comments=[]models.Comment,total=int}
// @Failure      400 {string} string "Invalid parameters"
// @Failure      403 {string} string "Forbidden"
// @Failure      404 {string} string "Post not found"
// @Failure      500 {string} string "Failed to get comments"
// @Security     BearerAuth
// @Router       /admin/posts/{slug}/comments [get]
func (h *AdminHandler) GetPostComments(w http.ResponseWriter, r *http.Request) {
	// URL에서 slug 추출
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "Post slug is required", http.StatusBadRequest)
		return
	}

	// Query 파라미터에서 site_id 추출
	siteIDStr := r.URL.Query().Get("site_id")
	if siteIDStr == "" {
		http.Error(w, "site_id query parameter is required", http.StatusBadRequest)
		return
	}
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid site_id", http.StatusBadRequest)
		return
	}

	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 사용자가 해당 사이트에 접근 권한이 있는지 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		http.Error(w, "Failed to check site access", http.StatusInternalServerError)
		return
	}

	if !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 포스트 조회
	post, err := database.GetPostBySlug(r.Context(), h.db, siteID, slug)
	if err != nil {
		http.Error(w, "Failed to get post", http.StatusInternalServerError)
		return
	}
	if post == nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// Query 파라미터에서 limit, offset 추출 (기본값: 50, 0)
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Admin용 댓글 조회 (삭제된 것 포함, IP 마스킹 없음)
	comments, total, err := database.GetAdminComments(r.Context(), h.db, post.ID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get comments", http.StatusInternalServerError)
		return
	}

	// Admin은 전체 IP와 마스킹된 IP 모두 볼 수 있음
	for _, comment := range comments {
		comment.IPAddressMasked = models.MaskIPAddress(comment.IPAddress)
		comment.IPAddressUnmasked = comment.IPAddress
		// 대댓글도 동일하게 처리
		for _, reply := range comment.Replies {
			reply.IPAddressMasked = models.MaskIPAddress(reply.IPAddress)
			reply.IPAddressUnmasked = reply.IPAddress
		}
	}

	// 응답 반환
	response := map[string]interface{}{
		"comments": comments,
		"total":    total,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
