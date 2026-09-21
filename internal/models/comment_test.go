package models

import (
	"strings"
	"testing"
)

// TestMaskIPAddress는 IP 주소 마스킹 결과를 테스트합니다
// IPv4는 앞 2 옥텟(/16), IPv6는 펼친 표기의 앞 2 그룹(/32)만 남겨야 합니다
func TestMaskIPAddress(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{name: "빈 문자열은 빈 문자열", ip: "", want: ""},
		{name: "IPv4", ip: "192.168.1.10", want: "192.168.***.***"},
		{name: "IPv4-mapped IPv6는 IPv4로 마스킹", ip: "::ffff:192.168.1.10", want: "192.168.***.***"},
		{name: "IPv6 압축 표기", ip: "2001:db8::1", want: "2001:0db8:****:****:****:****:****:****"},
		{name: "IPv6 중간 압축 표기", ip: "2001:0db8:85a3::8a2e:0370:7334", want: "2001:0db8:****:****:****:****:****:****"},
		{name: "IPv6 비압축 짧은 그룹", ip: "2001:db8:85a3:1:2:3:4:5", want: "2001:0db8:****:****:****:****:****:****"},
		{name: "IPv6 루프백", ip: "::1", want: "0000:0000:****:****:****:****:****:****"},
		{name: "IPv6 zone 포함", ip: "fe80::1%eth0", want: "fe80:0000:****:****:****:****:****:****"},
		{name: "파싱 불가 입력", ip: "not-an-ip", want: "***.***.***.***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: IP 마스킹
			got := MaskIPAddress(tt.ip)

			// Then: 기대한 마스킹 결과
			if got != tt.want {
				t.Errorf("MaskIPAddress(%q) = %q, want %q", tt.ip, got, tt.want)
			}
		})
	}
}

// TestMaskIPAddress_IPv6HidesAllButFirstTwoGroups는 IPv6 입력이 어떤 표기든
// 8개 그룹으로 펼쳐지고 앞 2 그룹 외에는 모두 가려지는지 테스트합니다
func TestMaskIPAddress_IPv6HidesAllButFirstTwoGroups(t *testing.T) {
	inputs := []string{
		"2001:db8::1",
		"2001:db8::",
		"::",
		"2001:db8:0:0:1::1",
		"2606:4700:4700::1111",
		"fe80::1%en0",
	}

	for _, ip := range inputs {
		t.Run(ip, func(t *testing.T) {
			// When: IP 마스킹
			groups := strings.Split(MaskIPAddress(ip), ":")

			// Then: 8개 그룹, 앞 2 그룹 이후는 모두 "****"
			if len(groups) != 8 {
				t.Fatalf("MaskIPAddress(%q) has %d groups, want 8", ip, len(groups))
			}
			for i, group := range groups[2:] {
				if group != "****" {
					t.Errorf("MaskIPAddress(%q) group %d = %q, want ****", ip, i+2, group)
				}
			}
		})
	}
}
