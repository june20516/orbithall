package models

import (
	"testing"
	"time"
)

// TestRefreshToken_IsExpired는 Refresh Token 만료 판단을 테스트합니다
func TestRefreshToken_IsExpired(t *testing.T) {
	expiresAt := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	token := &RefreshToken{ExpiresAt: expiresAt}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "만료 1초 전이면 유효", now: expiresAt.Add(-time.Second), want: false},
		{name: "만료 시각과 같으면 만료", now: expiresAt, want: true},
		{name: "만료 이후면 만료", now: expiresAt.Add(time.Second), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: 만료 여부 확인
			got := token.IsExpired(tt.now)

			// Then: 기대값과 일치
			if got != tt.want {
				t.Errorf("IsExpired(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}
