package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// LocalStorer writes files under BaseDir using the standard filesystem.
// Each Put creates a new file at BaseDir/key, copies src into it, and closes it.
// On any error after the file is created, the partial file is removed.
type LocalStorer struct {
	BaseDir string
}

func (s *LocalStorer) Put(_ context.Context, key string, src io.Reader, _ int64, _ string) (string, error) {
	full := filepath.Join(s.BaseDir, key)

	f, err := os.Create(full)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(f, src); err != nil {
		_ = f.Close()
		_ = os.Remove(full)
		return "", err
	}

	// f.Close() can technically fail (fsync error, ENOSPC, etc.), but it's
	// effectively unreachable in normal conditions and not worth a test.
	if err := f.Close(); err != nil {
		_ = os.Remove(full)
		return "", err
	}

	return full, nil
}
