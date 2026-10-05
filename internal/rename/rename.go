package rename

import (
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Renamer converts a client-supplied filename into a safe basename (without
// extension). The result must be filesystem-safe under both the local
// storer and any object-storage backend.
type Renamer interface {
	Rename(original string) string
}

// SanitizedRenamer keeps the original name's base but strips paths, the
// extension, and falls back to "default" for empty / dangerous inputs.
// This is the legacy behaviour preserved from the original getFileName().
type SanitizedRenamer struct{}

func (SanitizedRenamer) Rename(orig string) string {
	name := filepath.Base(orig)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "" || name == "." || name == "/" {
		return "default"
	}
	return name
}

// UUIDRenamer returns an opaque UUID-based name. Original filenames are
// discarded — useful when filenames may leak private info or be guessed.
type UUIDRenamer struct{}

func (UUIDRenamer) Rename(_ string) string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}
