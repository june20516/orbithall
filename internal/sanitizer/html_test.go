package sanitizer

import "testing"

func TestSanitizeComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain text should remain unchanged",
			input:    "This is a plain text comment",
			expected: "This is a plain text comment",
		},
		{
			name:     "Remove script tags",
			input:    "Hello <script>alert('XSS')</script> World",
			expected: "Hello  World",
		},
		{
			name:     "Remove all HTML tags",
			input:    "<b>Bold</b> and <i>italic</i> text",
			expected: "Bold and italic text",
		},
		{
			name:     "Remove dangerous onclick attribute",
			input:    "<div onclick=\"alert('XSS')\">Click me</div>",
			expected: "Click me",
		},
		{
			name:     "Remove iframe tags",
			input:    "Check this <iframe src=\"http://evil.com\"></iframe>",
			expected: "Check this ",
		},
		{
			name:     "Preserve Korean text",
			input:    "안녕하세요 <strong>반갑습니다</strong>",
			expected: "안녕하세요 반갑습니다",
		},
		{
			name:     "Handle empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Remove nested tags",
			input:    "<div><span><b>Nested</b></span></div>",
			expected: "Nested",
		},
		{
			name:     "Keep apostrophe as is",
			input:    "I'm happy",
			expected: "I'm happy",
		},
		{
			name:     "Keep ampersand as is",
			input:    "Tom & Jerry",
			expected: "Tom & Jerry",
		},
		{
			name:     "Keep double quotes as is",
			input:    `"quoted"`,
			expected: `"quoted"`,
		},
		{
			name:     "Keep less-than sign that is not a tag",
			input:    "1 < 2",
			expected: "1 < 2",
		},
		{
			name:     "Keep entity text typed literally by user",
			input:    "&amp; and &lt;b&gt;",
			expected: "&amp; and &lt;b&gt;",
		},
		{
			name:     "Remove script tag and keep following text",
			input:    "<script>alert(1)</script>hi",
			expected: "hi",
		},
		{
			name:     "Keep special characters around removed tags",
			input:    "<b>Tom</b> & <i>Jerry's</i>",
			expected: "Tom & Jerry's",
		},
		{
			name:     "Preserve Korean and emoji",
			input:    "안녕하세요 😀👍",
			expected: "안녕하세요 😀👍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeComment(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeComment(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
