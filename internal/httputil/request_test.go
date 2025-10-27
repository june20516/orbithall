package httputil

import (
	"net/http"
	"testing"
)

// TestGetIPAddress_XForwardedFor는 X-Forwarded-For 헤더에서 IP를 추출하는지 테스트합니다
func TestGetIPAddress_XForwardedFor(t *testing.T) {
	// Given: X-Forwarded-For 헤더가 포함된 요청
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
	req.RemoteAddr = "192.168.1.1:12345"

	// When: IP 주소 추출
	ip := GetIPAddress(req)

	// Then: X-Forwarded-For의 첫 번째 IP를 반환해야 함
	expected := "203.0.113.1"
	if ip != expected {
		t.Errorf("Expected %s, got %s", expected, ip)
	}
}

// TestGetIPAddress_XRealIP는 X-Real-IP 헤더에서 IP를 추출하는지 테스트합니다
func TestGetIPAddress_XRealIP(t *testing.T) {
	// Given: X-Real-IP 헤더가 포함된 요청 (X-Forwarded-For 없음)
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "203.0.113.5")
	req.RemoteAddr = "192.168.1.1:12345"

	// When: IP 주소 추출
	ip := GetIPAddress(req)

	// Then: X-Real-IP를 반환해야 함
	expected := "203.0.113.5"
	if ip != expected {
		t.Errorf("Expected %s, got %s", expected, ip)
	}
}

// TestGetIPAddress_RemoteAddr는 RemoteAddr에서 IP를 추출하는지 테스트합니다
func TestGetIPAddress_RemoteAddr(t *testing.T) {
	// Given: 헤더 없이 RemoteAddr만 있는 요청
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	// When: IP 주소 추출
	ip := GetIPAddress(req)

	// Then: RemoteAddr에서 포트를 제거한 IP를 반환해야 함
	expected := "192.168.1.100"
	if ip != expected {
		t.Errorf("Expected %s, got %s", expected, ip)
	}
}

// TestGetIPAddress_Priority는 헤더 우선순위를 테스트합니다
func TestGetIPAddress_Priority(t *testing.T) {
	// Given: 모든 헤더가 포함된 요청
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.Header.Set("X-Real-IP", "203.0.113.5")
	req.RemoteAddr = "192.168.1.1:12345"

	// When: IP 주소 추출
	ip := GetIPAddress(req)

	// Then: X-Forwarded-For가 최우선이어야 함
	expected := "203.0.113.1"
	if ip != expected {
		t.Errorf("Expected %s, got %s", expected, ip)
	}
}

// TestGetUserAgent는 User-Agent 헤더를 추출하는지 테스트합니다
func TestGetUserAgent(t *testing.T) {
	// Given: User-Agent 헤더가 포함된 요청
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")

	// When: User-Agent 추출
	userAgent := GetUserAgent(req)

	// Then: 정확한 User-Agent를 반환해야 함
	expected := "Mozilla/5.0 (Test Browser)"
	if userAgent != expected {
		t.Errorf("Expected %s, got %s", expected, userAgent)
	}
}

// TestGetUserAgent_Empty는 User-Agent가 없을 때 빈 문자열을 반환하는지 테스트합니다
func TestGetUserAgent_Empty(t *testing.T) {
	// Given: User-Agent 헤더가 없는 요청
	req, _ := http.NewRequest("GET", "/test", nil)

	// When: User-Agent 추출
	userAgent := GetUserAgent(req)

	// Then: 빈 문자열을 반환해야 함
	if userAgent != "" {
		t.Errorf("Expected empty string, got %s", userAgent)
	}
}
