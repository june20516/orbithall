package database

import "errors"

// ============================================
// Sentinel Errors
// ============================================
// database 계층에서 발생하는 도메인 에러들을 정의합니다.
// handlers에서 errors.Is()로 에러 타입을 확인할 수 있습니다.

var (
	// ErrParentCommentNotFound는 대댓글 생성 시 부모 댓글이 존재하지 않을 때 발생
	ErrParentCommentNotFound = errors.New("parent comment not found")

	// ErrNestedReplyNotAllowed는 2-depth 이상의 댓글 생성을 시도할 때 발생 (1-depth만 허용)
	ErrNestedReplyNotAllowed = errors.New("nested reply not allowed (max depth is 1)")

	// ErrCommentNotFound는 댓글 조회/수정/삭제 시 댓글이 존재하지 않을 때 발생
	ErrCommentNotFound = errors.New("comment not found")

	// ErrWrongPassword는 댓글 수정/삭제 시 비밀번호가 일치하지 않을 때 발생
	ErrWrongPassword = errors.New("wrong password")

	// ErrEditTimeExpired는 댓글 수정 가능 시간(30분)이 초과되었을 때 발생
	ErrEditTimeExpired = errors.New("edit time expired")
)
