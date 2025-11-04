package validators

import (
	"net/url"
	"strings"
)

// SiteCreateInput은 사이트 생성 시 입력 데이터 구조체
type SiteCreateInput struct {
	Name        string   `json:"name"`         // 사이트 이름 (필수, 1-100자)
	Domain      string   `json:"domain"`       // 사이트 도메인 (필수)
	CORSOrigins []string `json:"cors_origins"` // CORS 허용 오리진 목록 (필수, URL 형식)
}

// Validate는 사이트 생성 입력값을 검증
// name(필수, 1-100자), domain(필수), cors_origins(필수, URL 형식) 검증
func (s *SiteCreateInput) Validate() error {
	errors := make(ValidationErrors)

	// 사이트 이름 검증: 공백 제거 후 1-100자 확인
	name := strings.TrimSpace(s.Name)
	if name == "" {
		errors["name"] = "Name is required"
	} else if len(name) > 100 {
		errors["name"] = "Name must be 100 characters or less"
	}

	// 도메인 검증: 공백 제거 후 필수 확인
	domain := strings.TrimSpace(s.Domain)
	if domain == "" {
		errors["domain"] = "Domain is required"
	}

	// CORS origins 검증: 최소 1개 필수, 각 URL 형식 확인
	if len(s.CORSOrigins) == 0 {
		errors["cors_origins"] = "At least one CORS origin is required"
	} else {
		for i, origin := range s.CORSOrigins {
			if err := validateURL(origin); err != nil {
				errors["cors_origins"] = err.Error()
				break
			}
			// 중복 검사는 하지 않음 (DB에서 처리 가능)
			_ = i
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// SiteUpdateInput은 사이트 수정 시 입력 데이터 구조체
// 모든 필드가 포인터 타입: nil이면 수정하지 않음
type SiteUpdateInput struct {
	Name        *string   `json:"name"`         // 사이트 이름 (선택, 1-100자)
	CORSOrigins *[]string `json:"cors_origins"` // CORS 허용 오리진 목록 (선택, URL 형식)
	IsActive    *bool     `json:"is_active"`    // 활성화 상태 (선택)
}

// Validate는 사이트 수정 입력값을 검증
// name(선택, 1-100자), cors_origins(선택, URL 형식), is_active(선택) 검증
func (s *SiteUpdateInput) Validate() error {
	errors := make(ValidationErrors)

	// 사이트 이름 검증: 제공된 경우에만 1-100자 확인
	if s.Name != nil {
		name := strings.TrimSpace(*s.Name)
		if name == "" {
			errors["name"] = "Name cannot be empty"
		} else if len(name) > 100 {
			errors["name"] = "Name must be 100 characters or less"
		}
	}

	// CORS origins 검증: 제공된 경우 최소 1개 필수, 각 URL 형식 확인
	if s.CORSOrigins != nil {
		if len(*s.CORSOrigins) == 0 {
			errors["cors_origins"] = "At least one CORS origin is required"
		} else {
			for _, origin := range *s.CORSOrigins {
				if err := validateURL(origin); err != nil {
					errors["cors_origins"] = err.Error()
					break
				}
			}
		}
	}

	// IsActive는 bool 타입이므로 별도 검증 불필요

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// validateURL은 URL 형식을 검증하는 내부 헬퍼 함수
// http:// 또는 https:// 스키마가 있는지 확인
func validateURL(rawURL string) error {
	// URL 파싱
	u, err := url.Parse(rawURL)
	if err != nil {
		return ValidationErrors{"url": "Invalid URL format"}
	}

	// 스키마 확인: http 또는 https만 허용
	if u.Scheme != "http" && u.Scheme != "https" {
		return ValidationErrors{"url": "URL must have http:// or https:// scheme"}
	}

	// 호스트 확인: 호스트가 비어있으면 안 됨
	if u.Host == "" {
		return ValidationErrors{"url": "URL must have a valid host"}
	}

	return nil
}
