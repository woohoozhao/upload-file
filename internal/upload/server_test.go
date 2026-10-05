package upload

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"upload-file/internal/key"
	"upload-file/internal/rename"
	"upload-file/internal/storage"
)

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

// newTestServer wires a Server backed by MemStorer + slog.Default() so tests
// run without touching the filesystem.
func newTestServer(t *testing.T) (*Server, *storage.MemStorer) {
	t.Helper()
	storer := storage.NewMemStorer()
	srv := &Server{
		Storer:     storer,
		Renamer:    rename.SanitizedRenamer{},
		KeyBuilder: key.FlatKeyBuilder{Dir: "uploads"},
		Logger:     slog.Default(),
		MaxBytes:   10 << 20,
	}
	return srv, storer
}

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

// failingStorer is a storage.Storer whose Put always returns an error.
type failingStorer struct{}

func (failingStorer) Put(_ context.Context, _ string, _ io.Reader, _ int64, _ string) (string, error) {
	return "", errStorerBoom
}

var errStorerBoom = stringError("storer boom")

type stringError string

func (e stringError) Error() string { return string(e) }

// ----------------- Server.HandleUpload -----------------

func TestHandleUpload(t *testing.T) {
	t.Run("valid jpeg uploads", func(t *testing.T) {
		srv, storer := newTestServer(t)

		req := buildMultipart(t, "avatar", "test.jpg", jpegBytes)
		rr := httptest.NewRecorder()
		srv.HandleUpload(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "upload ok") {
			t.Errorf("body should contain 'upload ok', got %q", rr.Body.String())
		}
		if storer.Len() != 1 {
			t.Fatalf("expected 1 stored file, got %d", storer.Len())
		}

		keys := storer.Keys()
		if len(keys) != 1 {
			t.Fatalf("expected exactly 1 key, got %d: %v", len(keys), keys)
		}
		key := keys[0]
		data, ok := storer.Get(key)
		if !ok {
			t.Fatalf("Get(%q) returned ok=false", key)
		}
		if !strings.HasPrefix(key, "uploads/test_") {
			t.Errorf("key should start with 'uploads/test_', got %q", key)
		}
		if !strings.HasSuffix(key, ".jpg") {
			t.Errorf("key should end with '.jpg', got %q", key)
		}
		if len(data) < 512 || !bytes.Equal(data[:512], jpegBytes) {
			t.Errorf("stored bytes do not match input (first 512)")
		}
	})

	t.Run("wrong field name falls through to EOF and returns 200", func(t *testing.T) {
		srv, storer := newTestServer(t)

		req := buildMultipart(t, "wrong_field", "test.jpg", jpegBytes)
		rr := httptest.NewRecorder()
		srv.HandleUpload(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (EOF path)", rr.Code)
		}
		if storer.Len() != 0 {
			t.Errorf("nothing should have been stored, got %d", storer.Len())
		}
	})

	t.Run("non-image content returns 415", func(t *testing.T) {
		srv, storer := newTestServer(t)

		req := buildMultipart(t, "avatar", "fake.jpg", []byte("not actually an image"))
		rr := httptest.NewRecorder()
		srv.HandleUpload(rr, req)
		if rr.Code != http.StatusUnsupportedMediaType {
			t.Errorf("status = %d, want 415", rr.Code)
		}
		if storer.Len() != 0 {
			t.Errorf("nothing should have been stored on mime rejection, got %d", storer.Len())
		}
	})

	t.Run("invalid multipart body returns 400", func(t *testing.T) {
		srv, storer := newTestServer(t)

		req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("not multipart"))
		req.Header.Set("Content-Type", "text/plain")

		rr := httptest.NewRecorder()
		srv.HandleUpload(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
		if storer.Len() != 0 {
			t.Errorf("nothing should have been stored on bad request, got %d", storer.Len())
		}
	})

	t.Run("Storer error surfaces as 500", func(t *testing.T) {
		srv, storer := newTestServer(t)
		srv.Storer = failingStorer{}

		req := buildMultipart(t, "avatar", "boom.jpg", jpegBytes)
		rr := httptest.NewRecorder()
		srv.HandleUpload(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", rr.Code)
		}
		if storer.Len() != 0 {
			t.Errorf("nothing should have been stored when storer failed, got %d", storer.Len())
		}
	})

	t.Run("exceeding MaxBytes returns 400", func(t *testing.T) {
		srv, storer := newTestServer(t)
		srv.MaxBytes = 100 // very small cap; jpegBytes (512) exceeds it

		req := buildMultipart(t, "avatar", "big.jpg", jpegBytes)
		rr := httptest.NewRecorder()
		srv.HandleUpload(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
		if storer.Len() != 0 {
			t.Errorf("nothing should have been stored when over limit, got %d", storer.Len())
		}
	})

	t.Run("nil Logger falls back to slog.Default()", func(t *testing.T) {
		srv, _ := newTestServer(t)
		srv.Logger = nil // explicitly unset

		req := buildMultipart(t, "avatar", "fallback.jpg", jpegBytes)
		rr := httptest.NewRecorder()
		srv.HandleUpload(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (nil Logger should still work)", rr.Code)
		}
	})

	t.Run("NextPart parse error returns 400", func(t *testing.T) {
		// Content-Type is multipart (so MultipartReader() succeeds) but
		// the body never contains the boundary. mr.NextPart() then fails
		// with a non-EOF parse error, exercising the uncovered branch.
		req := httptest.NewRequest(http.MethodPost, "/upload",
			strings.NewReader("this is not multipart at all"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")

		srv, storer := newTestServer(t)
		rr := httptest.NewRecorder()
		srv.HandleUpload(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
		if storer.Len() != 0 {
			t.Errorf("nothing should have been stored on NextPart error, got %d", storer.Len())
		}
	})

	t.Run("part read error other than EOF returns 400", func(t *testing.T) {
		// Build a multipart request whose body errors out partway through
		// with something other than io.EOF / io.ErrUnexpectedEOF, so that
		// the io.ReadFull error branch in HandleUpload is exercised.
		body := &brokenMultipart{}
		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set("Content-Type", body.contentType())

		srv, storer := newTestServer(t)
		rr := httptest.NewRecorder()
		srv.HandleUpload(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rr.Code)
		}
		if storer.Len() != 0 {
			t.Errorf("nothing should have been stored on read error, got %d", storer.Len())
		}
	})
}

// brokenMultipart emits a syntactically valid multipart header for "avatar"
// then fails the part-data read with a non-EOF error, forcing
// io.ReadFull to surface a generic read error inside HandleUpload.
type brokenMultipart struct {
	written int
	ctype   string
}

func (b *brokenMultipart) Read(p []byte) (int, error) {
	const header = "--xyz\r\n" +
		"Content-Disposition: form-data; name=\"avatar\"; filename=\"a.jpg\"\r\n" +
		"Content-Type: image/jpeg\r\n\r\n"
	if b.written < len(header) {
		n := copy(p, header[b.written:])
		b.written += n
		return n, nil
	}
	return 0, errBrokenRead
}

var errBrokenRead = stringError("simulated read failure")

func (b *brokenMultipart) contentType() string {
	return "multipart/form-data; boundary=xyz"
}
