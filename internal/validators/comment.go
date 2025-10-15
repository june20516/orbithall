package validators

import (
	"fmt"
	"strings"
)

// ValidationErrors는 필드명과 에러 메시지를 매핑하는 타입
// 여러 검증 에러를 하나의 에러로 모아서 반환할 때 사용
type ValidationErrors map[string]string

// Error는 error 인터페이스를 구현
// 모든 검증 에러를 쉼표로 구분된 문자열로 반환
func (v ValidationErrors) Error() string {
	var messages []string
	for field, msg := range v {
		messages = append(messages, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(messages, ", ")
}

// CommentCreateInput은 댓글 생성 시 입력 데이터 구조체
type CommentCreateInput struct {
	AuthorName string // 작성자 이름
	Password   string // 비밀번호 (수정/삭제 시 사용)
	Content    string // 댓글 내용
	ParentID   *int   // 대댓글인 경우 부모 댓글 ID (선택)
}

// Validate는 댓글 생성 입력값을 검증
// author_name(1-100자), password(4-50자), content(1-10000자), parent_id(양수) 검증
func (c *CommentCreateInput) Validate() error {
	errors := make(ValidationErrors)

	// 작성자 이름 검증: 공백 제거 후 1-100자 확인
	authorName := strings.TrimSpace(c.AuthorName)
	if authorName == "" {
		errors["author_name"] = "Author name is required"
	} else if len(authorName) > 100 {
		errors["author_name"] = "Author name must be 100 characters or less"
	}

	// 비밀번호 검증: 4-50자
	if len(c.Password) < 4 {
		errors["password"] = "Password must be at least 4 characters"
	} else if len(c.Password) > 50 {
		errors["password"] = "Password must be 50 characters or less"
	}

	// 내용 검증: 공백 제거 후 1-10000자 확인
	if strings.TrimSpace(c.Content) == "" {
		errors["content"] = "Content is required"
	} else if len(c.Content) > 10000 {
		errors["content"] = "Content must be 10000 characters or less"
	}

	// 부모 ID 검증: 대댓글인 경우 양의 정수여야 함
	if c.ParentID != nil && *c.ParentID <= 0 {
		errors["parent_id"] = "Parent ID must be a positive integer"
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// CommentUpdateInput은 댓글 수정 시 입력 데이터 구조체
type CommentUpdateInput struct {
	Password string // 비밀번호 (인증용)
	Content  string // 수정할 댓글 내용
}

// Validate는 댓글 수정 입력값을 검증
// password(필수), content(1-10000자) 검증
func (c *CommentUpdateInput) Validate() error {
	errors := make(ValidationErrors)

	// 비밀번호 검증: 필수
	if c.Password == "" {
		errors["password"] = "Password is required"
	}

	// 내용 검증: 공백 제거 후 1-10000자 확인
	if strings.TrimSpace(c.Content) == "" {
		errors["content"] = "Content is required"
	} else if len(c.Content) > 10000 {
		errors["content"] = "Content must be 10000 characters or less"
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// CommentDeleteInput은 댓글 삭제 시 입력 데이터 구조체
type CommentDeleteInput struct {
	Password string // 비밀번호 (인증용)
}

// Validate는 댓글 삭제 입력값을 검증
// password(필수) 검증
func (c *CommentDeleteInput) Validate() error {
	errors := make(ValidationErrors)

	// 비밀번호 검증: 필수
	if c.Password == "" {
		errors["password"] = "Password is required"
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}
