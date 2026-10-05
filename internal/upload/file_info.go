package upload

import (
	"fmt"
	"net/http"
)

var mimeExtMap = map[string]string{
	"image/jpeg": ".jpg",
}

// getFileExt sniffs MIME from the first bytes of buf and resolves it to an
// extension using mimeExtMap. Add more entries to mimeExtMap to support
// additional formats.
func getFileExt(buf []byte) (string, error) {
	mimeType := http.DetectContentType(buf)
	ext, ok := mimeExtMap[mimeType]
	if !ok {
		return "", fmt.Errorf("unknown mime type: %s", mimeType)
	}
	return ext, nil
}
