package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
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
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Security     BearerAuth
// @Router       /admin/sites [get]
func (h *AdminHandler) ListSites(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
		return
	}

	// 사용자의 사이트 목록 조회
	sites, err := database.GetUserSites(r.Context(), h.db, user.ID)
	if err != nil {
		log.Printf("[ERROR] admin ListSites: 사이트 목록 조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get sites", nil)
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
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Failure      404 {object} ErrorResponse "SITE_NOT_FOUND"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Security     BearerAuth
// @Router       /admin/sites/{id} [get]
func (h *AdminHandler) GetSite(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
		return
	}

	// URL 파라미터에서 site_id 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid site ID", nil)
		return
	}

	// 사이트 조회
	site, err := database.GetSiteByID(r.Context(), h.db, siteID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, ErrSiteNotFound, "Site not found", nil)
			return
		}
		log.Printf("[ERROR] admin GetSite: 사이트 조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get site", nil)
		return
	}

	// 접근 권한 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		log.Printf("[ERROR] admin GetSite: 접근 권한 확인: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to check access", nil)
		return
	}

	if !hasAccess {
		respondError(w, http.StatusNotFound, ErrSiteNotFound, "Site not found", nil)
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
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Security     BearerAuth
// @Router       /admin/sites [post]
func (h *AdminHandler) CreateSite(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
		return
	}

	// Content-Type 검증
	if r.Header.Get("Content-Type") != "application/json" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Content-Type must be application/json", nil)
		return
	}

	// JSON 요청 파싱
	var input validators.SiteCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid JSON", nil)
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
		log.Printf("[ERROR] admin CreateSite: 사이트 생성: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to create site", nil)
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
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Failure      403 {object} ErrorResponse "FORBIDDEN"
// @Failure      404 {object} ErrorResponse "SITE_NOT_FOUND"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Security     BearerAuth
// @Router       /admin/sites/{id} [put]
func (h *AdminHandler) UpdateSite(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
		return
	}

	// URL 파라미터에서 site_id 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid site ID", nil)
		return
	}

	// Content-Type 검증
	if r.Header.Get("Content-Type") != "application/json" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Content-Type must be application/json", nil)
		return
	}

	// JSON 요청 파싱
	var input validators.SiteUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid JSON", nil)
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
		log.Printf("[ERROR] admin UpdateSite: 접근 권한 확인: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to check access", nil)
		return
	}

	if !hasAccess {
		respondError(w, http.StatusForbidden, ErrForbidden, "Forbidden", nil)
		return
	}

	// 기존 사이트 조회 (현재 값 가져오기)
	site, err := database.GetSiteByID(r.Context(), h.db, siteID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, ErrSiteNotFound, "Site not found", nil)
			return
		}
		log.Printf("[ERROR] admin UpdateSite: 사이트 조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get site", nil)
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
			respondError(w, http.StatusNotFound, ErrSiteNotFound, "Site not found", nil)
			return
		}
		log.Printf("[ERROR] admin UpdateSite: 사이트 수정: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to update site", nil)
		return
	}

	// 수정된 사이트 재조회
	updatedSite, err := database.GetSiteByID(r.Context(), h.db, siteID)
	if err != nil {
		log.Printf("[ERROR] admin UpdateSite: 수정된 사이트 재조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get updated site", nil)
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
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Failure      403 {object} ErrorResponse "FORBIDDEN"
// @Failure      404 {object} ErrorResponse "SITE_NOT_FOUND"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Security     BearerAuth
// @Router       /admin/sites/{id} [delete]
func (h *AdminHandler) DeleteSite(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
		return
	}

	// URL 파라미터에서 site_id 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid site ID", nil)
		return
	}

	// 접근 권한 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		log.Printf("[ERROR] admin DeleteSite: 접근 권한 확인: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to check access", nil)
		return
	}

	if !hasAccess {
		respondError(w, http.StatusForbidden, ErrForbidden, "Forbidden", nil)
		return
	}

	// 사이트 삭제
	err = database.DeleteSite(r.Context(), h.db, siteID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, ErrSiteNotFound, "Site not found", nil)
			return
		}
		log.Printf("[ERROR] admin DeleteSite: 사이트 삭제: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to delete site", nil)
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
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Security     BearerAuth
// @Router       /admin/profile [get]
func (h *AdminHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
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
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Failure      403 {object} ErrorResponse "FORBIDDEN"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Security     BearerAuth
// @Router       /admin/sites/{id}/stats [get]
func (h *AdminHandler) GetSiteStats(w http.ResponseWriter, r *http.Request) {
	// URL에서 site ID 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid site ID", nil)
		return
	}

	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
		return
	}

	// 사용자가 해당 사이트에 접근 권한이 있는지 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		log.Printf("[ERROR] admin GetSiteStats: 접근 권한 확인: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to check site access", nil)
		return
	}

	if !hasAccess {
		respondError(w, http.StatusForbidden, ErrForbidden, "Forbidden", nil)
		return
	}

	// 통계 조회
	stats, err := database.GetSiteStats(r.Context(), h.db, siteID)
	if err != nil {
		log.Printf("[ERROR] admin GetSiteStats: 통계 조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get site stats", nil)
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
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Failure      403 {object} ErrorResponse "FORBIDDEN"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Security     BearerAuth
// @Router       /admin/sites/{id}/posts [get]
func (h *AdminHandler) ListSitePosts(w http.ResponseWriter, r *http.Request) {
	// URL에서 site ID 추출
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid site ID", nil)
		return
	}

	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
		return
	}

	// 사용자가 해당 사이트에 접근 권한이 있는지 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		log.Printf("[ERROR] admin ListSitePosts: 접근 권한 확인: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to check site access", nil)
		return
	}

	if !hasAccess {
		respondError(w, http.StatusForbidden, ErrForbidden, "Forbidden", nil)
		return
	}

	// 포스트 목록 조회
	posts, err := database.ListPostsBySite(r.Context(), h.db, siteID)
	if err != nil {
		log.Printf("[ERROR] admin ListSitePosts: 포스트 목록 조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get posts", nil)
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
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Failure      403 {object} ErrorResponse "FORBIDDEN"
// @Failure      404 {object} ErrorResponse "POST_NOT_FOUND"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Security     BearerAuth
// @Router       /admin/posts/{slug}/comments [get]
func (h *AdminHandler) GetPostComments(w http.ResponseWriter, r *http.Request) {
	// URL에서 slug 추출
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Post slug is required", nil)
		return
	}

	// Query 파라미터에서 site_id 추출
	siteIDStr := r.URL.Query().Get("site_id")
	if siteIDStr == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "site_id query parameter is required", nil)
		return
	}
	siteID, err := strconv.ParseInt(siteIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid site_id", nil)
		return
	}

	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
		return
	}

	// 사용자가 해당 사이트에 접근 권한이 있는지 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, siteID)
	if err != nil {
		log.Printf("[ERROR] admin GetPostComments: 접근 권한 확인: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to check site access", nil)
		return
	}

	if !hasAccess {
		respondError(w, http.StatusForbidden, ErrForbidden, "Forbidden", nil)
		return
	}

	// 포스트 조회
	post, err := database.GetPostBySlug(r.Context(), h.db, siteID, slug)
	if err != nil {
		log.Printf("[ERROR] admin GetPostComments: 포스트 조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get post", nil)
		return
	}
	if post == nil {
		respondError(w, http.StatusNotFound, ErrPostNotFound, "Post not found", nil)
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
		log.Printf("[ERROR] admin GetPostComments: 댓글 조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get comments", nil)
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

// DeleteComment는 사이트 소유자가 댓글을 soft delete합니다
// 이미 삭제된 댓글도 204를 반환합니다 (멱등)
// @Summary      댓글 삭제
// @Description  사이트 소유자가 댓글을 삭제합니다 (soft delete). 대댓글은 유지되며, 이미 삭제된 댓글도 204를 반환합니다
// @Tags         admin
// @Produce      plain
// @Param        id path int true "Comment ID"
// @Success      204 "No Content"
// @Failure      400 {object} ErrorResponse "INVALID_INPUT"
// @Failure      401 {object} ErrorResponse "MISSING_TOKEN, INVALID_TOKEN, EXPIRED_TOKEN, USER_NOT_FOUND, UNAUTHORIZED"
// @Failure      403 {object} ErrorResponse "FORBIDDEN"
// @Failure      404 {object} ErrorResponse "COMMENT_NOT_FOUND"
// @Failure      500 {object} ErrorResponse "INTERNAL_SERVER_ERROR"
// @Security     BearerAuth
// @Router       /admin/comments/{id} [delete]
func (h *AdminHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	// Context에서 사용자 추출
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		respondError(w, http.StatusUnauthorized, ErrUnauthorized, "Unauthorized", nil)
		return
	}

	// URL 파라미터에서 comment_id 추출
	commentID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || commentID <= 0 {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid comment ID", nil)
		return
	}

	// 댓글 조회 (삭제된 댓글 포함)
	comment, err := database.GetCommentByID(r.Context(), h.db, commentID)
	if err != nil {
		log.Printf("[ERROR] admin DeleteComment: 댓글 조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get comment", nil)
		return
	}
	if comment == nil {
		respondError(w, http.StatusNotFound, ErrCommentNotFound, "Comment not found", nil)
		return
	}

	// 댓글이 속한 사이트 확인 (댓글은 FK로 포스트에 묶여 있으므로 포스트가 없으면 서버 오류)
	post, err := database.GetPostByID(r.Context(), h.db, comment.PostID)
	if err != nil || post == nil {
		log.Printf("[ERROR] admin DeleteComment: 포스트 조회: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get post", nil)
		return
	}

	// 접근 권한 확인
	hasAccess, err := database.HasUserSiteAccess(r.Context(), h.db, user.ID, post.SiteID)
	if err != nil {
		log.Printf("[ERROR] admin DeleteComment: 접근 권한 확인: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to check site access", nil)
		return
	}
	if !hasAccess {
		respondError(w, http.StatusForbidden, ErrForbidden, "Forbidden", nil)
		return
	}

	// 이미 삭제된 댓글은 권한 확인 후에 멱등 처리
	if comment.IsDeleted {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 삭제 (동시 요청으로 먼저 삭제된 경우도 ErrCommentNotFound로 오므로 성공 처리)
	err = database.DeleteComment(r.Context(), h.db, commentID)
	if err != nil && !errors.Is(err, database.ErrCommentNotFound) {
		log.Printf("[ERROR] admin DeleteComment: 댓글 삭제: %v", err)
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to delete comment", nil)
		return
	}

	// 204 No Content 응답
	w.WriteHeader(http.StatusNoContent)
}
