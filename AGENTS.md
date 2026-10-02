# AGENTS.md

## What this is
Single-file Go HTTP server (`upload-file` module). `POST /upload` accepts an avatar as `multipart/form-data` and saves it to the working directory. Test client is [file.html](file.html).

Everything lives in [main.go](main.go) — no subpackages, no tests.

## Build & run
- `go run .` — listens on `:80`, **requires root** on macOS/Linux. For local dev, change the `:80` in `main` ([main.go:154-166](main.go#L154-L166)) to e.g. `":8080"` and update the form `action` in [file.html](file.html).
- No `go test` target yet.

## Architecture
Two handlers coexist in [main.go](main.go); only one is wired up at a time in `main`:

- `handleUploadStream` — **active**. Streams parts via `req.MultipartReader()`. Constant memory regardless of file size.
- `handleUpload` — buffered via `req.FormFile()`. Currently commented out in `main`. Loads the whole file into memory.

Shared helpers:
- `getFileName` — strip path + extension from the client filename; fall back to `"default"` for empty results.
- `getFileExt` — sniff MIME from the first 512 bytes via `http.DetectContentType`, then map to an extension via `mimeExtMap`.
- `setFilePath` — builds `./{name}_{unix_seconds}{ext}`. Uses `os.Create` (truncates if exists).

Logging is `log/slog` throughout. Don't introduce `log.Printf`.

## Known pitfalls
- **`mimeExtMap` only contains `image/jpeg`** ([main.go](main.go)). The HTML form advertises `png/jpeg/gif`, so any non-jpeg upload returns `400 unknown mime type: …`. The HTML `accept` attribute is client-side only — to accept more types, extend `mimeExtMap`.
- **No upload size limit.** `MultipartReader` will consume whatever the client sends. Add `http.MaxBytesReader` before exposing beyond localhost.
- **Filename collisions within the same second.** `setFilePath` uses `time.Now().Unix()` (1-second resolution). Two uploads of the same filename in the same second overwrite each other.
- **Response `Write` errors are logged, not returned.** After a successful upload, `Write()` errors to `http.ResponseWriter` are logged via `slog.Error` and dropped. Intentional — the upload already committed.
