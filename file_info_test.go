package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// ----------------- getFileName -----------------

func TestGetFileName(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFileName(tt.input)
			if got != tt.want {
				t.Errorf("getFileName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ----------------- getFileExt -----------------

func TestGetFileExt(t *testing.T) {
	// Minimal JPEG SOI + APP0/JFIF header — http.DetectContentType needs 512 bytes
	// to be fully accurate, so we pad the input.
	jpeg := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00,
		0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, 0xFF, 0xD9,
	}

	t.Run("jpeg detected", func(t *testing.T) {
		buf := make([]byte, 512)
		copy(buf, jpeg)
		ext, err := getFileExt(buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ext != ".jpg" {
			t.Errorf("getFileExt(jpeg) = %q, want .jpg", ext)
		}
	})

	t.Run("plain text rejected", func(t *testing.T) {
		buf := make([]byte, 512)
		copy(buf, []byte("Hello, this is plain text not an image at all."))
		_, err := getFileExt(buf)
		if err == nil {
			t.Fatal("expected error for non-image content")
		}
		if !strings.Contains(err.Error(), "unknown mime type") {
			t.Errorf("error should mention 'unknown mime type', got: %v", err)
		}
	})

	t.Run("empty buffer rejected", func(t *testing.T) {
		_, err := getFileExt([]byte{})
		if err == nil {
			t.Fatal("expected error for empty buffer")
		}
	})
}

// ----------------- setFilePath -----------------

func TestSetFilePath(t *testing.T) {
	t.Run("uses default dir when path empty", func(t *testing.T) {
		before := time.Now().Unix()
		path := setFilePath("", "photo", ".jpg")
		after := time.Now().Unix()

		if !strings.HasPrefix(path, "./uploads/photo_") {
			t.Errorf("path should start with './uploads/photo_', got %q", path)
		}
		if !strings.HasSuffix(path, ".jpg") {
			t.Errorf("path should end with '.jpg', got %q", path)
		}

		tsStr := strings.TrimPrefix(path, "./uploads/photo_")
		tsStr = strings.TrimSuffix(tsStr, ".jpg")
		var ts int64
		if _, err := fmt.Sscanf(tsStr, "%d", &ts); err != nil {
			t.Fatalf("failed to parse timestamp from %q: %v", tsStr, err)
		}
		if ts < before || ts > after {
			t.Errorf("timestamp %d not in [%d, %d]", ts, before, after)
		}
	})

	t.Run("uses custom dir", func(t *testing.T) {
		path := setFilePath("/tmp/custom", "doc", ".pdf")
		if !strings.HasPrefix(path, "/tmp/custom/doc_") {
			t.Errorf("path should start with '/tmp/custom/doc_', got %q", path)
		}
		if !strings.HasSuffix(path, ".pdf") {
			t.Errorf("path should end with '.pdf', got %q", path)
		}
	})
}
