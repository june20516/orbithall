package handlers

import (
	"net/http"
	"strconv"

	"github.com/june20516/orbithall/internal/httputil"
)

// ============================================
// HTTP 요청 관련 공통 헬퍼 함수
// ============================================

// GetIPAddress는 httputil.GetIPAddress를 호출합니다
// 하위 호환성을 위해 유지되며, 새 코드는 httputil.GetIPAddress를 직접 사용하는 것을 권장합니다
func GetIPAddress(r *http.Request) string {
	return httputil.GetIPAddress(r)
}

// GetUserAgent는 httputil.GetUserAgent를 호출합니다
// 하위 호환성을 위해 유지되며, 새 코드는 httputil.GetUserAgent를 직접 사용하는 것을 권장합니다
func GetUserAgent(r *http.Request) string {
	return httputil.GetUserAgent(r)
}

// ParseInt64Param은 문자열을 int64로 파싱합니다
// URL 파라미터를 숫자로 변환할 때 사용합니다
func ParseInt64Param(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

// ParseQueryInt는 쿼리 파라미터를 int로 파싱하며 기본값을 지원합니다
// 파라미터가 없거나 파싱 실패 시 기본값을 반환합니다
func ParseQueryInt(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return parsed
}
