package logger

import (
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"short", "hello world", "hello world"},
		{"at-limit", strings.Repeat("a", MaxLogMessageLength), strings.Repeat("a", MaxLogMessageLength)},
		{"over-limit", strings.Repeat("a", MaxLogMessageLength+100),
			strings.Repeat("a", MaxLogMessageLength) + "...[truncated]"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitize(tc.in)
			if got != tc.want {
				t.Errorf("sanitize(len=%d) 长度=%d 期望=%d", len(tc.in), len(got), len(tc.want))
			}
		})
	}
}

func TestMaxLogMessageLength(t *testing.T) {
	if MaxLogMessageLength <= 0 {
		t.Fatal("MaxLogMessageLength 应为正数")
	}
	if MaxLogMessageLength > 64*1024 {
		t.Fatal("MaxLogMessageLength 不应过大，避免无意义截断")
	}
}
