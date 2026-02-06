package organizer

import (
	"strings"
	"testing"
)

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CON", "CON_safe"},
		{"PRN", "PRN_safe"},
		{"AUX", "AUX_safe"},
		{"NUL", "NUL_safe"},
		{"COM1", "COM1_safe"},
		{"LPT1", "LPT1_safe"},
		{"my<file>", "my_file_"},
		{"folder/path", "folder_path"},
		{"file:name", "file_name"},
		{"  trim  ", "trim"},
		{"", "Unknown"},
		{".", "Unknown"},
		{"..", "Unknown"},
		{"a" + strings.Repeat("b", 150), "a" + strings.Repeat("b", 99)}, // Length limit test
	}
	for _, tt := range tests {
		got := SanitizeFolderName(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeFolderName(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}
