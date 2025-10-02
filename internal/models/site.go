package models

import "time"

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
