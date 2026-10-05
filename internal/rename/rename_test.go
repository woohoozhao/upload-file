package rename

import (
	"strings"
	"testing"
)

func TestSanitizedRenamer(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal jpg", "photo.jpg", "photo"},
		{"no extension", "photo", "photo"},
		{"path traversal", "../../../etc/passwd", "passwd"},
		{"hidden file", ".hidden", "default"},
		{"empty string", "", "default"},
		{"just dot", ".", "default"},
		{"just slash", "/", "default"},
		{"multiple dots", "my.file.tar.gz", "my.file.tar"},
		{"nested path", "a/b/c.png", "c"},
		{"filename with space", "my photo.jpg", "my photo"},
	}

	r := SanitizedRenamer{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Rename(tt.input)
			if got != tt.want {
				t.Errorf("SanitizedRenamer.Rename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUUIDRenamer(t *testing.T) {
	r := UUIDRenamer{}
	out := r.Rename("anything.jpg")
	if len(out) != 32 {
		t.Errorf("UUIDRenamer output should be 32 hex chars (no dashes), got %q (len %d)", out, len(out))
	}
	if strings.Contains(out, "-") {
		t.Errorf("UUIDRenamer output should not contain dashes, got %q", out)
	}
	if r.Rename("x") == out {
		t.Error("UUIDRenamer should return unique values across calls")
	}
}
