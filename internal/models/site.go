package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Site는 Orbithall을 사용하는 사이트 정보를 나타냅니다
// 멀티 테넌시 지원을 위해 각 사이트를 구분하고 인증합니다
type Site struct {
	// Name은 사이트의 이름입니다 (예: "코드버스 블로그")
	Name string `json:"name"`

	// Domain은 사이트의 도메인입니다 (예: "blog.codeverse.com")
	Domain string `json:"domain"`

	// APIKey는 API 인증에 사용되는 UUID 키입니다
	// 클라이언트는 X-Orbithall-API-Key 헤더로 이 값을 전송합니다
	APIKey string `json:"api_key"`

	// CORSOrigins는 CORS 허용 도메인 목록입니다
	// PostgreSQL의 TEXT[] 타입과 매핑됩니다
	// 예: ["http://localhost:3000", "https://blog.example.com"]
	CORSOrigins []string `json:"cors_origins"`

	// IsActive는 사이트의 활성화 여부입니다
	// false인 경우 API 접근이 차단됩니다
	IsActive bool `json:"is_active"`

	// 메타데이터
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SiteStats는 사이트의 통계 정보를 나타냅니다
type SiteStats struct {
	PostCount           int `json:"post_count"`
	CommentCount        int `json:"comment_count"`
	DeletedCommentCount int `json:"deleted_comment_count"`
}

// GenerateAPIKey는 주어진 prefix로 API 키를 생성합니다
//
// prefix 종류:
//   - "orb_live_": 프로덕션 환경용
//   - "orb_test_": 테스트 환경용
//
// 생성 방식:
//   - 12 바이트 랜덤 데이터 생성
//   - hex 인코딩하여 24자 문자열로 변환
//   - prefix와 결합
//
// 예시:
//   - GenerateAPIKey("orb_live_") → "orb_live_a3f5c8d9e2b1f4c6a8d7e9f3b2c5a8d7"
//   - GenerateAPIKey("orb_test_") → "orb_test_1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e"
func GenerateAPIKey(prefix string) string {
	// 12 바이트 = 24 hex 문자
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		// crypto/rand는 시스템 난수 생성기를 사용하므로
		// 실패는 심각한 시스템 레벨 문제를 의미 (panic 적절)
		panic("failed to generate random bytes for API key: " + err.Error())
	}
	return prefix + hex.EncodeToString(bytes)
}
