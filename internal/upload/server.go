package upload

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"upload-file/internal/key"
	"upload-file/internal/rename"
	"upload-file/internal/storage"
)

// fileExtSize is the byte budget used for MIME sniffing and the first
// chunk read from each upload part. 512 matches http.DetectContentType's
// recommended minimum for accurate detection.
const fileExtSize = 512

// Server holds the upload dependencies and exposes the HTTP handler.
// All collaborators are interfaces, so tests can swap any of them out.
type Server struct {
	Storer     storage.Storer
	Renamer    rename.Renamer
	KeyBuilder key.KeyBuilder
	Logger     *slog.Logger // nil falls back to slog.Default() at call sites
	MaxBytes   int64        // per-upload byte limit; MaxBytesReader is applied automatically
}

// logger returns the configured logger or slog.Default() if unset.
// Centralised so the handler doesn't need to nil-check at every call site.
func (s *Server) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

// HandleUpload is the streaming multipart upload handler.
// It sniffs the first fileExtSize bytes for MIME, then pipes the rest
// through the configured Storer under a key produced by KeyBuilder.
func (s *Server) HandleUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Cap the request body so a malicious client can't stream forever.
	r.Body = http.MaxBytesReader(w, r.Body, s.MaxBytes+fileExtSize)

	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			// No avatar field — preserve legacy behaviour of returning 200 "upload ok".
			// See AGENTS.md for context; flag for review.
			_, _ = w.Write([]byte("upload ok"))
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if part.FormName() != "avatar" {
			_ = part.Close()
			continue
		}

		// Sniff MIME from the first fileExtSize bytes.
		head := make([]byte, fileExtSize)
		n, err := io.ReadFull(part, head)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			_ = part.Close()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mimeType := http.DetectContentType(head[:n])
		fileExt, err := getFileExt(head[:n])
		if err != nil {
			_ = part.Close()
			http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			return
		}

		base := s.Renamer.Rename(part.FileName())
		key := s.KeyBuilder.Build(base, fileExt)

		// Splice the head bytes back in front of the remaining stream so
		// Storer receives the complete content.
		body := io.MultiReader(bytes.NewReader(head[:n]), part)

		savedKey, err := s.Storer.Put(r.Context(), key, body, -1, mimeType)
		_ = part.Close()
		if err != nil {
			s.logger().Error("upload failed", "key", key, "mime", mimeType, "err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.logger().Info("upload ok", "key", savedKey, "mime", mimeType)
		_, _ = w.Write([]byte("upload ok"))
		return
	}
}
