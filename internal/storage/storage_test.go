package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ----------------- MemStorer -----------------

func TestMemStorer_PutAndGet(t *testing.T) {
	m := NewMemStorer()
	ctx := context.Background()

	got, err := m.Put(ctx, "k1", bytes.NewReader([]byte("hello")), 5, "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if got != "k1" {
		t.Errorf("Put returned key %q, want %q", got, "k1")
	}

	data, ok := m.Get("k1")
	if !ok {
		t.Fatal("Get(k1) returned ok=false")
	}
	if string(data) != "hello" {
		t.Errorf("Get(k1) = %q, want %q", data, "hello")
	}
}

func TestMemStorer_OverwriteKey(t *testing.T) {
	m := NewMemStorer()
	ctx := context.Background()

	if _, err := m.Put(ctx, "k", bytes.NewReader([]byte("first")), 5, ""); err != nil {
		t.Fatalf("first Put: %v", err)
	}
	if _, err := m.Put(ctx, "k", bytes.NewReader([]byte("second")), 6, ""); err != nil {
		t.Fatalf("second Put: %v", err)
	}

	data, _ := m.Get("k")
	if string(data) != "second" {
		t.Errorf("after overwrite Get = %q, want %q", data, "second")
	}
	if m.Len() != 1 {
		t.Errorf("Len = %d, want 1 (overwrite should not add a new entry)", m.Len())
	}
}

func TestMemStorer_KeysAndLen(t *testing.T) {
	m := NewMemStorer()
	ctx := context.Background()

	for _, k := range []string{"a", "b", "c"} {
		if _, err := m.Put(ctx, k, bytes.NewReader([]byte("x")), 1, ""); err != nil {
			t.Fatalf("Put(%q): %v", k, err)
		}
	}

	keys := m.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys returned %d entries, want 3: %v", len(keys), keys)
	}

	if m.Len() != 3 {
		t.Errorf("Len = %d, want 3", m.Len())
	}
}

func TestMemStorer_PutPropagatesReaderError(t *testing.T) {
	// Covers the io.ReadAll error path in MemStorer.Put.
	m := NewMemStorer()
	_, err := m.Put(context.Background(), "k", &erringReader{err: io.ErrUnexpectedEOF}, -1, "")
	if err == nil {
		t.Fatal("expected error from Put with failing reader")
	}
}

func TestMemStorer_ConcurrentPut(t *testing.T) {
	// Run with `go test -race` to verify the mutex actually protects the map.
	m := NewMemStorer()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			k := string(rune('a' + (i % 26)))
			_, _ = m.Put(ctx, k, bytes.NewReader([]byte("x")), 1, "")
		}(i)
	}
	wg.Wait()

	if m.Len() > 26 {
		t.Errorf("Len = %d, want ≤ 26 (only lowercase letters used)", m.Len())
	}
}

// ----------------- LocalStorer -----------------

func TestLocalStorer_PutWritesFile(t *testing.T) {
	dir := t.TempDir()
	s := &LocalStorer{BaseDir: dir}

	saved, err := s.Put(context.Background(), "doc.txt", bytes.NewReader([]byte("body")), 4, "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	want := filepath.Join(dir, "doc.txt")
	if saved != want {
		t.Errorf("saved key = %q, want %q", saved, want)
	}

	got, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "body" {
		t.Errorf("file contents = %q, want %q", got, "body")
	}
}

func TestLocalStorer_PutCleansUpOnCopyError(t *testing.T) {
	// Force a copy error by handing Put a reader that errors on Read.
	dir := t.TempDir()
	s := &LocalStorer{BaseDir: dir}

	errReader := &erringReader{err: io.ErrUnexpectedEOF}
	_, err := s.Put(context.Background(), "boom.bin", errReader, -1, "")
	if err == nil {
		t.Fatal("expected error from Put with failing reader")
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		// Partial file should have been removed.
		if strings.HasPrefix(e.Name(), "boom") {
			t.Errorf("partial file %q was not cleaned up after Put failure", e.Name())
		}
	}
}

func TestLocalStorer_PutFailsWhenDirMissing(t *testing.T) {
	s := &LocalStorer{BaseDir: filepath.Join(t.TempDir(), "does-not-exist")}

	_, err := s.Put(context.Background(), "x", bytes.NewReader([]byte("y")), 1, "")
	if err == nil {
		t.Fatal("expected error when BaseDir does not exist")
	}
}

// erringReader is an io.Reader that always returns the configured error.
type erringReader struct{ err error }

func (r *erringReader) Read(_ []byte) (int, error) { return 0, r.err }
