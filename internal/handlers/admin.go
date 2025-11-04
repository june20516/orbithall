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
