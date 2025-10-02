package models

import "time"

// Post는 블로그 포스트의 메타데이터를 나타냅니다
// 실제 포스트 내용은 Next.js 블로그에 있으며, 이 테이블은 댓글 연결용입니다
type Post struct {
	// SiteID는 이 포스트가 속한 사이트의 ID입니다
	// 멀티 테넌시 지원을 위해 사이트별로 포스트를 구분합니다
	// 데이터베이스: sites 테이블에 대한 외래키 (ON DELETE CASCADE)
	SiteID int64 `json:"site_id"`

	// Slug는 포스트의 URL slug입니다 (예: "how-to-use-go")
	// 데이터베이스: (site_id, slug) 조합이 unique 제약조건
	Slug string `json:"slug"`

	// Title은 포스트의 제목입니다
	Title string `json:"title"`

	// CommentCount는 이 포스트에 달린 댓글 수입니다
	// 캐시 역할을 하며, 댓글 추가/삭제 시 업데이트됩니다
	CommentCount int `json:"comment_count"`

	// 메타데이터
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
