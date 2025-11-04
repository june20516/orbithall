package validators

import (
	"testing"
)

// TestSiteCreateInput_Validate는 사이트 생성 입력값 검증 테스트
func TestSiteCreateInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   SiteCreateInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "유효한 입력 - 성공",
			input: SiteCreateInput{
				Name:        "My Blog",
				Domain:      "blog.example.com",
				CORSOrigins: []string{"https://blog.example.com"},
			},
			wantErr: false,
		},
		{
			name: "유효한 입력 - 여러 CORS origins",
			input: SiteCreateInput{
				Name:        "My Blog",
				Domain:      "blog.example.com",
				CORSOrigins: []string{"https://blog.example.com", "http://localhost:3000"},
			},
			wantErr: false,
		},
		{
			name: "name 누락 - 실패",
			input: SiteCreateInput{
				Name:        "",
				Domain:      "blog.example.com",
				CORSOrigins: []string{"https://blog.example.com"},
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "name 공백만 - 실패",
			input: SiteCreateInput{
				Name:        "   ",
				Domain:      "blog.example.com",
				CORSOrigins: []string{"https://blog.example.com"},
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "name 너무 긺 (100자 초과) - 실패",
			input: SiteCreateInput{
				Name:        "a" + string(make([]byte, 100)), // 101자
				Domain:      "blog.example.com",
				CORSOrigins: []string{"https://blog.example.com"},
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "domain 누락 - 실패",
			input: SiteCreateInput{
				Name:        "My Blog",
				Domain:      "",
				CORSOrigins: []string{"https://blog.example.com"},
			},
			wantErr: true,
			errMsg:  "domain",
		},
		{
			name: "domain 공백만 - 실패",
			input: SiteCreateInput{
				Name:        "My Blog",
				Domain:      "   ",
				CORSOrigins: []string{"https://blog.example.com"},
			},
			wantErr: true,
			errMsg:  "domain",
		},
		{
			name: "cors_origins 빈 배열 - 실패",
			input: SiteCreateInput{
				Name:        "My Blog",
				Domain:      "blog.example.com",
				CORSOrigins: []string{},
			},
			wantErr: true,
			errMsg:  "cors_origins",
		},
		{
			name: "cors_origins nil - 실패",
			input: SiteCreateInput{
				Name:        "My Blog",
				Domain:      "blog.example.com",
				CORSOrigins: nil,
			},
			wantErr: true,
			errMsg:  "cors_origins",
		},
		{
			name: "cors_origins 잘못된 URL 형식 - 실패",
			input: SiteCreateInput{
				Name:        "My Blog",
				Domain:      "blog.example.com",
				CORSOrigins: []string{"not-a-url"},
			},
			wantErr: true,
			errMsg:  "cors_origins",
		},
		{
			name: "cors_origins 스키마 없음 - 실패",
			input: SiteCreateInput{
				Name:        "My Blog",
				Domain:      "blog.example.com",
				CORSOrigins: []string{"blog.example.com"},
			},
			wantErr: true,
			errMsg:  "cors_origins",
		},
		{
			name: "cors_origins 하나는 유효, 하나는 무효 - 실패",
			input: SiteCreateInput{
				Name:        "My Blog",
				Domain:      "blog.example.com",
				CORSOrigins: []string{"https://blog.example.com", "invalid"},
			},
			wantErr: true,
			errMsg:  "cors_origins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 에러 메시지에 특정 필드명이 포함되어야 함
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

// TestSiteUpdateInput_Validate는 사이트 수정 입력값 검증 테스트
func TestSiteUpdateInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   SiteUpdateInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "유효한 입력 - 모든 필드 수정",
			input: SiteUpdateInput{
				Name:        strPtr("Updated Name"),
				CORSOrigins: &[]string{"https://newdomain.com"},
				IsActive:    boolPtr(false),
			},
			wantErr: false,
		},
		{
			name: "유효한 입력 - name만 수정",
			input: SiteUpdateInput{
				Name: strPtr("Updated Name"),
			},
			wantErr: false,
		},
		{
			name: "유효한 입력 - cors_origins만 수정",
			input: SiteUpdateInput{
				CORSOrigins: &[]string{"https://newdomain.com", "http://localhost:3000"},
			},
			wantErr: false,
		},
		{
			name: "유효한 입력 - is_active만 수정",
			input: SiteUpdateInput{
				IsActive: boolPtr(true),
			},
			wantErr: false,
		},
		{
			name: "유효한 입력 - 모든 필드 nil (수정 없음)",
			input: SiteUpdateInput{
				Name:        nil,
				CORSOrigins: nil,
				IsActive:    nil,
			},
			wantErr: false,
		},
		{
			name: "name 빈 문자열 - 실패",
			input: SiteUpdateInput{
				Name: strPtr(""),
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "name 공백만 - 실패",
			input: SiteUpdateInput{
				Name: strPtr("   "),
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "name 너무 긺 (100자 초과) - 실패",
			input: SiteUpdateInput{
				Name: strPtr("a" + string(make([]byte, 100))), // 101자
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "cors_origins 빈 배열 - 실패",
			input: SiteUpdateInput{
				CORSOrigins: &[]string{},
			},
			wantErr: true,
			errMsg:  "cors_origins",
		},
		{
			name: "cors_origins 잘못된 URL 형식 - 실패",
			input: SiteUpdateInput{
				CORSOrigins: &[]string{"not-a-url"},
			},
			wantErr: true,
			errMsg:  "cors_origins",
		},
		{
			name: "cors_origins 하나는 유효, 하나는 무효 - 실패",
			input: SiteUpdateInput{
				CORSOrigins: &[]string{"https://valid.com", "invalid"},
			},
			wantErr: true,
			errMsg:  "cors_origins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 에러 메시지에 특정 필드명이 포함되어야 함
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

// 헬퍼 함수: 문자열이 특정 부분 문자열을 포함하는지 확인
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// 헬퍼 함수: 문자열 포인터 생성
func strPtr(s string) *string {
	return &s
}

// 헬퍼 함수: bool 포인터 생성
func boolPtr(b bool) *bool {
	return &b
}
