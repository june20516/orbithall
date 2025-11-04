package models

import (
	"net"
	"strings"
	"time"
)

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

	// IPAddressMasked는 마스킹된 IP 주소입니다
	// API 응답에 포함되며, 개인정보 보호를 위해 마지막 옥텟이 가려집니다 (예: 192.168.1.xxx)
	// 데이터베이스에 저장되지 않고, 런타임에서만 생성됩니다
	IPAddressMasked string `json:"ip_address_masked,omitempty"`

	// IPAddressUnmasked는 마스킹되지 않은 전체 IP 주소입니다 (Admin 전용)
	// Admin API에서만 포함되며, 일반 API 응답에서는 제거됩니다
	// 데이터베이스에 저장되지 않고, 런타임에서만 생성됩니다
	IPAddressUnmasked string `json:"ip_address_unmasked,omitempty"`

	// Replies는 이 댓글에 달린 대댓글 목록입니다
	// 계층적 댓글 구조를 표현하기 위해 사용됩니다
	// 데이터베이스에 저장되지 않고, 쿼리 결과를 조합하여 생성됩니다
	Replies []*Comment `json:"replies,omitempty"`

	// 메타데이터
	ID        int64      `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// MaskIPAddress는 IP 주소를 마스킹하여 개인정보를 보호합니다
// IPv4: 앞 2 옥텟만 표시 (예: 192.168.***.*** )
// IPv6: 앞 4개 그룹만 표시 (예: 2001:0db8:****:****:****:****:****:****)
func MaskIPAddress(ip string) string {
	// 빈 문자열 처리
	if ip == "" {
		return ""
	}

	// IP 파싱
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		// 파싱 실패 시 기본 마스킹 (안전한 폴백)
		return "***.***.***.***"
	}

	// IPv4 처리
	if parsedIP.To4() != nil {
		parts := strings.Split(ip, ".")
		if len(parts) == 4 {
			return parts[0] + "." + parts[1] + ".***.***"
		}
		return "***.***.***.***"
	}

	// IPv6 처리
	parts := strings.Split(ip, ":")
	if len(parts) >= 4 {
		return strings.Join(parts[:4], ":") + ":****:****:****:****"
	}

	return "****:****:****:****:****:****:****:****"
}
