package models

import "time"

// User는 Google OAuth 로그인을 통해 인증된 Admin 사용자 정보를 나타냅니다
type User struct {
	// Email은 사용자의 이메일 주소입니다 (Google 계정)
	Email string `json:"email"`

	// Name은 사용자의 이름입니다 (Google 프로필)
	Name string `json:"name"`

	// PictureURL은 사용자의 프로필 이미지 URL입니다
	PictureURL string `json:"picture_url,omitempty"`

	// GoogleID는 Google OAuth의 고유 식별자입니다 (내부 전용)
	// API 응답에서는 제외됩니다
	GoogleID string `json:"-"`

	// 메타데이터
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
