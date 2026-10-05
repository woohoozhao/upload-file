package main

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"upload-file/internal/storage"
)

// buildStorer should always return a LocalStorer pointing at localUploadDir,
// regardless of UPLOAD_BACKEND (the OSS branch has been removed until
// real OSS credentials are wired up).
func TestBuildStorer(t *testing.T) {
	t.Setenv("UPLOAD_BACKEND", "")
	s := buildStorer()

	ls, ok := s.(*storage.LocalStorer)
	if !ok {
		t.Fatalf("buildStorer returned %T, want *storage.LocalStorer", s)
	}
	if !strings.HasSuffix(ls.BaseDir, "uploads") {
		t.Errorf("BaseDir = %q, want to end with 'uploads'", ls.BaseDir)
	}
}

func TestRun_GracefulShutdownViaContext(t *testing.T) {
	t.Chdir(t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	exited := 0
	exit := func(code int) { exited = code }

	// ":0" → kernel picks a free port; we just want a successful bind.
	err := run(ctx, "127.0.0.1:0", exit)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if exited != 0 {
		t.Errorf("exit called with code %d, want 0 (graceful path)", exited)
	}
}

func TestRun_MkdirFailureExits(t *testing.T) {
	// Make ./uploads a regular file so MkdirAll fails.
	tmp := t.TempDir()
	t.Chdir(tmp)

	if err := os.WriteFile(localUploadDir, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	exited := 0
	exit := func(code int) { exited = code }

	err := run(context.Background(), "127.0.0.1:0", exit)
	if err == nil {
		t.Fatal("expected *exitError from mkdir failure, got nil")
	}
	if _, ok := err.(*exitError); !ok {
		t.Errorf("err type = %T, want *exitError", err)
	}
	if exited != 1 {
		t.Errorf("exit code = %d, want 1", exited)
	}
}

func TestRun_ListenFailureExits(t *testing.T) {
	t.Chdir(t.TempDir())

	// Bind a listener to a port and hold it — the kernel-assigned address
	// becomes "already in use" for run's second listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("setup listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	exited := 0
	exit := func(code int) { exited = code }

	runErr := run(context.Background(), addr, exit)
	if runErr == nil {
		t.Fatal("expected *exitError from listen failure, got nil")
	}
	if _, ok := runErr.(*exitError); !ok {
		t.Errorf("err type = %T, want *exitError", runErr)
	}
	if exited != 1 {
		t.Errorf("exit code = %d, want 1", exited)
	}
}

// (no helpers — uses os.WriteFile directly)
