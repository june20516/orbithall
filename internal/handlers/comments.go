package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/june20516/orbithall/internal/database"
	"github.com/june20516/orbithall/internal/models"
	"github.com/june20516/orbithall/internal/sanitizer"
	"github.com/june20516/orbithall/internal/validators"
)

// ============================================
// 상수
// ============================================

// EditTimeLimit은 댓글 수정 가능 시간 제한입니다 (30분)
const EditTimeLimit = 30 * time.Minute

// ============================================
// CommentHandler 구조체
// ============================================

// CommentHandler는 댓글 관련 HTTP 요청을 처리합니다
// 댓글 생성, 조회, 수정, 삭제 기능을 제공합니다
type CommentHandler struct {
	db *sql.DB
}

// NewCommentHandler는 CommentHandler의 새 인스턴스를 생성합니다
// 데이터베이스 연결을 주입받아 의존성을 관리합니다
func NewCommentHandler(db *sql.DB) *CommentHandler {
	return &CommentHandler{
		db: db,
	}
}

// ============================================
// 댓글 전용 헬퍼 함수
// ============================================

// filterDeletedCommentsAndMaskIP는 삭제된 댓글을 필터링하고 모든 댓글의 IP를 마스킹합니다
//
// 삭제된 댓글의 필터링 규칙 (Soft Delete 방식):
// - 대댓글이 있는 삭제된 댓글: 계층 구조 유지를 위해 응답에 포함
//   (author_name과 content는 빈 문자열, isDeleted=true로 클라이언트가 판단)
// - 대댓글이 없는 삭제된 댓글: 응답 배열에서 완전히 제거
//
// IP 마스킹: 모든 댓글의 IP 주소를 부분 마스킹 (예: 192.168.***.***)
func filterDeletedCommentsAndMaskIP(comments []*models.Comment) []*models.Comment {
	filtered := make([]*models.Comment, 0, len(comments))

	for _, comment := range comments {
		if comment.IsDeleted {
			// 삭제된 댓글에 대댓글이 있으면 계층 구조 유지를 위해 포함
			if len(comment.Replies) > 0 {
				// 삭제된 댓글의 내용은 비움 (클라이언트가 isDeleted 플래그로 판단)
				comment.AuthorName = ""
				comment.Content = ""
				comment.IPAddressMasked = models.MaskIPAddress(comment.IPAddress)

				// 대댓글들의 IP 마스킹
				for j := range comment.Replies {
					comment.Replies[j].IPAddressMasked = models.MaskIPAddress(comment.Replies[j].IPAddress)
				}

				filtered = append(filtered, comment)
			}
			// 대댓글이 없는 삭제된 댓글은 배열에서 완전히 제거
		} else {
			// 삭제되지 않은 댓글: IP 마스킹만 수행
			comment.IPAddressMasked = models.MaskIPAddress(comment.IPAddress)

			// 대댓글들의 IP 마스킹
			for j := range comment.Replies {
				comment.Replies[j].IPAddressMasked = models.MaskIPAddress(comment.Replies[j].IPAddress)
			}

			filtered = append(filtered, comment)
		}
	}

	return filtered
}

// ============================================
// HTTP 핸들러 메서드
// ============================================

// CreateComment는 새 댓글을 생성합니다
// POST /api/posts/:slug/comments
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	// 1. Context에서 사이트 정보 추출 (AuthMiddleware에서 주입됨)
	site := GetSiteFromContext(r.Context())
	if site == nil {
		respondError(w, http.StatusUnauthorized, ErrMissingAPIKey, "Site not found in context", nil)
		return
	}

	// 2. URL 파라미터에서 slug 추출
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Post slug is required", nil)
		return
	}

	// 3. 요청 본문 파싱
	var input validators.CommentCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Invalid request body", nil)
		return
	}

	// 4. 입력 검증
	if err := input.Validate(); err != nil {
		validationErrs, ok := err.(validators.ValidationErrors)
		if ok {
			respondError(w, http.StatusBadRequest, ErrInvalidInput, "Validation failed", validationErrs)
		} else {
			respondError(w, http.StatusBadRequest, ErrInvalidInput, err.Error(), nil)
		}
		return
	}

	// 5. HTML 새니타이제이션 (XSS 방어)
	input.Content = sanitizer.SanitizeComment(input.Content)
	input.AuthorName = sanitizer.SanitizeComment(input.AuthorName)

	// 6. 포스트 가져오기 또는 생성 (slug를 title로도 사용)
	post, err := database.GetOrCreatePost(h.db, site.ID, slug, slug)
	if err != nil {
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get or create post", nil)
		return
	}

	// 7. parent_id가 있으면 int64로 변환
	var parentID *int64
	if input.ParentID != nil {
		pid := int64(*input.ParentID)
		parentID = &pid
	}

	// 8. IP 주소 및 User-Agent 추출
	ipAddress := GetIPAddress(r)
	userAgent := GetUserAgent(r)

	// 9. 댓글 생성 (database.CreateComment가 2-depth 검증 및 비밀번호 해싱 처리)
	comment, err := database.CreateComment(h.db, post.ID, parentID, input.AuthorName, input.Password, input.Content, ipAddress, userAgent)
	if err != nil {
		// Sentinel errors를 사용한 에러 타입 확인
		if errors.Is(err, database.ErrNestedReplyNotAllowed) {
			respondError(w, http.StatusBadRequest, ErrInvalidInput, "Nested replies are not allowed (max depth is 1)", nil)
			return
		}
		if errors.Is(err, database.ErrParentCommentNotFound) {
			respondError(w, http.StatusNotFound, ErrCommentNotFound, "Parent comment not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to create comment", nil)
		return
	}

	// 10. 댓글 카운트 증가
	if err := database.IncrementCommentCount(h.db, post.ID); err != nil {
		// 카운트 증가 실패는 로깅만 하고 계속 진행 (댓글은 이미 생성됨)
		// TODO: 로깅 추가
	}

	// 11. IP 주소 마스킹
	comment.IPAddressMasked = models.MaskIPAddress(comment.IPAddress)

	// 12. 201 Created 응답 (비밀번호 해시 제외)
	response := map[string]interface{}{
		"id":                comment.ID,
		"post_id":           comment.PostID,
		"parent_id":         comment.ParentID,
		"author_name":       comment.AuthorName,
		"content":           comment.Content,
		"ip_address_masked": comment.IPAddressMasked,
		"is_deleted":        comment.IsDeleted,
		"created_at":        comment.CreatedAt,
		"updated_at":        comment.UpdatedAt,
		"deleted_at":        comment.DeletedAt,
	}

	respondJSON(w, http.StatusCreated, response)
}

// ListComments는 포스트의 댓글 목록을 조회합니다
// GET /api/posts/:slug/comments
func (h *CommentHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	// 1. Context에서 사이트 정보 추출
	site := GetSiteFromContext(r.Context())
	if site == nil {
		respondError(w, http.StatusUnauthorized, ErrMissingAPIKey, "Site not found in context", nil)
		return
	}

	// 2. URL 파라미터에서 slug 추출
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		respondError(w, http.StatusBadRequest, ErrInvalidInput, "Post slug is required", nil)
		return
	}

	// 3. 쿼리 파라미터 파싱 (page, limit)
	page := ParseQueryInt(r, "page", 1)
	limit := ParseQueryInt(r, "limit", 50)

	// 유효성 검증
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	// 4. 포스트 조회 (없으면 빈 배열 반환)
	post, err := database.GetPostBySlug(h.db, site.ID, slug)
	if err != nil {
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to get post", nil)
		return
	}

	// 포스트가 없으면 빈 배열 반환
	if post == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"comments": []models.Comment{},
			"pagination": map[string]interface{}{
				"current_page":   page,
				"total_pages":    0,
				"total_comments": 0,
				"per_page":       limit,
			},
		})
		return
	}

	// 5. offset 계산
	offset := (page - 1) * limit

	// 6. 댓글 목록 조회 (2-level 트리 구조)
	comments, totalCount, err := database.ListComments(h.db, post.ID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, ErrInternalServer, "Failed to list comments", nil)
		return
	}

	// 7. 삭제된 댓글 필터링 및 IP 마스킹
	// (대댓글 있으면 빈 값으로 포함, 없으면 제거)
	comments = filterDeletedCommentsAndMaskIP(comments)

	// 8. 페이지네이션 계산
	totalPages := (totalCount + limit - 1) / limit

	// 9. 응답
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"comments": comments,
		"pagination": map[string]interface{}{
			"current_page":   page,
			"total_pages":    totalPages,
			"total_comments": totalCount,
			"per_page":       limit,
		},
	})
}

// UpdateComment는 기존 댓글을 수정합니다
// PUT /api/comments/:id
func (h *CommentHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	// TODO: 구현 예정
	respondJSON(w, http.StatusOK, map[string]string{"message": "UpdateComment - TODO"})
}

// DeleteComment는 댓글을 삭제합니다 (soft delete)
// DELETE /api/comments/:id
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	// TODO: 구현 예정
	respondJSON(w, http.StatusOK, map[string]string{"message": "DeleteComment - TODO"})
}
