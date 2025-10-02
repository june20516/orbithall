package models

import "time"

// Comment는 블로그 포스트에 달린 댓글을 나타냅니다
type Comment struct {
	// PostID는 이 댓글이 속한 포스트의 ID입니다
	// 데이터베이스: posts 테이블에 대한 외래키 (ON DELETE CASCADE)
	PostID int64 `json:"post_id"`

	// ParentID는 대댓글인 경우 부모 댓글의 ID입니다
	// nil이면 최상위 댓글, 값이 있으면 대댓글입니다
	// 데이터베이스: comments 테이블 자기 참조 외래키 (ON DELETE CASCADE)
	ParentID *int64 `json:"parent_id,omitempty"`

	// AuthorName은 댓글 작성자의 이름입니다
	AuthorName string `json:"author_name"`

	// AuthorPassword는 댓글 수정/삭제 시 사용하는 비밀번호입니다
	// bcrypt로 해시되어 저장되며, API 응답에는 포함되지 않습니다
	AuthorPassword string `json:"-"`

	// Content는 댓글의 본문 내용입니다
	Content string `json:"content"`

	// IsDeleted는 소프트 삭제 여부입니다
	// true인 경우 "삭제된 댓글입니다" 같은 메시지로 표시됩니다
	IsDeleted bool `json:"is_deleted"`

	// IPAddress는 댓글 작성자의 IP 주소입니다
	// 스팸 방지 목적으로 저장하며, API 응답에는 포함되지 않습니다
	// 데이터베이스: INET 타입
	IPAddress string `json:"-"`

	// UserAgent는 댓글 작성 시 사용한 브라우저 정보입니다
	// 스팸 방지 목적으로 저장하며, API 응답에는 포함되지 않습니다
	UserAgent string `json:"-"`

	// 메타데이터
	ID        int64      `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
