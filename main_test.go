package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain ensures ./uploads exists before any test runs.
// (The real binary creates it in main(); tests don't call main().)
func TestMain(m *testing.M) {
	if err := os.MkdirAll(deftPath, 0777); err != nil {
		panic("failed to create uploads dir: " + err.Error())
	}
	os.Exit(m.Run())
}

// cleanUploads removes every file inside deftPath. Tests in this file are
// destructive to the uploads/ directory — do not run against a populated one.
func cleanUploads(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(deftPath)
	if err != nil {
		return
	}
	for _, e := range entries {
		_ = os.Remove(filepath.Join(deftPath, e.Name()))
	}
}

// jpegBytes is a minimal JFIF blob (SOI + APP0/JFIF + EOI), padded to 512 bytes.
var jpegBytes = func() []byte {
	core := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00,
		0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, 0xFF, 0xD9,
	}
	buf := make([]byte, 512)
	copy(buf, core)
	return buf
}()

// buildMultipart constructs a multipart/form-data POST request with one file field.
func buildMultipart(t *testing.T, field, filename string, content []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

// ----------------- handleUpload (buffered) -----------------

func TestHandleUpload(t *testing.T) {
	cleanUploads(t)
	t.Cleanup(func() { cleanUploads(t) })

	t.Run("valid jpeg uploads", func(t *testing.T) {
		req := buildMultipart(t, "avatar", "test.jpg", jpegBytes)
		rr := httptest.NewRecorder()
		handleUpload(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "upload ok") {
			t.Errorf("body should contain 'upload ok', got %q", rr.Body.String())
		}

		entries, _ := os.ReadDir(deftPath)
		if len(entries) != 1 {
			t.Fatalf("expected 1 file in uploads, got %d", len(entries))
		}
		if filepath.Ext(entries[0].Name()) != ".jpg" {
			t.Errorf("uploaded file should have .jpg ext, got %q", entries[0].Name())
		}
	})

	t.Run("missing avatar field returns 400", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		_ = w.Close()
		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", w.FormDataContentType())

		rr := httptest.NewRecorder()
		handleUpload(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("non-image content returns 400", func(t *testing.T) {
		req := buildMultipart(t, "avatar", "fake.jpg", []byte("not actually an image"))
		rr := httptest.NewRecorder()
		handleUpload(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})
}

// ----------------- handleUploadStream (streaming) -----------------

func TestHandleUploadStream(t *testing.T) {
	cleanUploads(t)
	t.Cleanup(func() { cleanUploads(t) })

	t.Run("valid jpeg streams", func(t *testing.T) {
		req := buildMultipart(t, "avatar", "stream.jpg", jpegBytes)
		rr := httptest.NewRecorder()
		handleUploadStream(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "upload ok") {
			t.Errorf("body should contain 'upload ok', got %q", rr.Body.String())
		}

		entries, _ := os.ReadDir(deftPath)
		if len(entries) != 1 {
			t.Fatalf("expected 1 file in uploads, got %d", len(entries))
		}
	})

	t.Run("wrong field name falls through to EOF and returns 200", func(t *testing.T) {
		// Stream handler skips non-"avatar" parts and writes "upload ok" on EOF.
		// This documents current behaviour — flag if you want it changed.
		req := buildMultipart(t, "wrong_field", "test.jpg", jpegBytes)
		rr := httptest.NewRecorder()
		handleUploadStream(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (EOF path)", rr.Code)
		}
	})

	t.Run("non-image content returns 415", func(t *testing.T) {
		req := buildMultipart(t, "avatar", "fake.jpg", []byte("not actually an image"))
		rr := httptest.NewRecorder()
		handleUploadStream(rr, req)
		if rr.Code != http.StatusUnsupportedMediaType {
			t.Errorf("status = %d, want 415", rr.Code)
		}
	})

	t.Run("invalid multipart body returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("not multipart"))
		req.Header.Set("Content-Type", "text/plain")

		rr := httptest.NewRecorder()
		handleUploadStream(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
	})
}
