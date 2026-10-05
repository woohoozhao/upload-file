package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"upload-file/internal/key"
	"upload-file/internal/rename"
	"upload-file/internal/storage"
	"upload-file/internal/upload"
)

// localUploadDir is the directory used by the default LocalStorer.
// Kept here (not in upload package) so cmd/server owns process-wide config.
const localUploadDir = "./uploads"

// exitError is returned by run() when it would otherwise call os.Exit.
// Tests assert on this instead of letting the process die.
type exitError struct{ Code int }

func (e *exitError) Error() string { return "exit code " + strconv.Itoa(e.Code) }

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	err := run(ctx, ":8080", os.Exit)
	if e, ok := err.(*exitError); ok {
		os.Exit(e.Code)
	}
}

// run is the testable core of the program. It returns *exitError when the
// caller should terminate; the exit function is injected so tests can
// observe (and suppress) the exit call.
//
// The function starts the server in a goroutine and returns once ctx is
// done (production: cancelled via signal handler; tests: cancel via t.Context).
func run(ctx context.Context, addr string, exit func(int)) (err error) {
	if err := os.MkdirAll(localUploadDir, 0750); err != nil {
		slog.Error("mkdir fail", "dir", localUploadDir, "err", err.Error())
		exit(1)
		return &exitError{Code: 1}
	}

	storer := buildStorer()

	srv := &upload.Server{
		Storer:     storer,
		Renamer:    rename.SanitizedRenamer{},
		KeyBuilder: key.FlatKeyBuilder{Dir: ""},
		Logger:     slog.Default(),
		MaxBytes:   10 << 20, // 10 MiB
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", srv.HandleUpload)

	httpSrv := &http.Server{Addr: addr, Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		_ = httpSrv.Shutdown(context.Background())
		return nil
	case err := <-errCh:
		if err != nil {
			slog.Error("listen", "err", err.Error())
			exit(1)
			return &exitError{Code: 1}
		}
		return nil
	}
}

// buildStorer picks the Storer implementation based on the UPLOAD_BACKEND
// env var. Defaults to local disk.
//
// To enable OSS: install github.com/aliyun/aliyun-oss-go-sdk/oss, then
// replace the default branch with the OSSStorer wiring in storage/oss.go.
func buildStorer() storage.Storer {
	// TODO: implement env-driven backend selection once OSS creds are
	// wired up in buildStorer. Until then, always local.
	_ = os.Getenv("UPLOAD_BACKEND") // hook for future use
	return &storage.LocalStorer{BaseDir: localUploadDir}
}
