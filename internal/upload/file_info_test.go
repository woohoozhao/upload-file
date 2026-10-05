package upload

import (
	"strings"
	"testing"
)

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
