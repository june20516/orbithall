package httputil

import (
	"net/http"
	"strings"
)

// GetIPAddress는 HTTP 요청에서 클라이언트의 실제 IP 주소를 추출합니다
// 프록시나 로드 밸런서를 거친 경우를 고려하여 다음 순서로 확인합니다:
// 1. X-Forwarded-For (프록시 체인의 첫 번째 IP)
// 2. X-Real-IP (프록시가 설정한 실제 IP)
// 3. RemoteAddr (직접 연결된 클라이언트 IP)
func GetIPAddress(r *http.Request) string {
	// X-Forwarded-For 헤더 확인 (쉼표로 구분된 IP 리스트)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// 첫 번째 IP가 실제 클라이언트 IP
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// X-Real-IP 헤더 확인
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// RemoteAddr 사용 (포트 번호 제거)
	// RemoteAddr 형식: "192.168.1.1:12345"
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

// GetUserAgent는 HTTP 요청에서 User-Agent 헤더를 추출합니다
// User-Agent는 클라이언트의 브라우저/앱 정보를 포함합니다
func GetUserAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}
