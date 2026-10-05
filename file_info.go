package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var mimeExtMap = map[string]string{
	"image/jpeg": ".jpg",
}

func getFileExt(buf []byte) (string, error) {
	mimeType := http.DetectContentType(buf)
	ext, ok := mimeExtMap[mimeType]
	if !ok {
		return "", fmt.Errorf("unknown mime type: %s", mimeType)
	}

	return ext, nil
}

func getFileName(filename string) string {
	name := filepath.Base(filename)

	name = strings.TrimSuffix(name, filepath.Ext(name))

	if name == "" || name == "." || name == "/" {
		return "default"
	}

	return name
}

const deftPath = "./uploads"

func setFilePath(path, fileName, fileExt string) string {
	if path == "" {
		path = deftPath
	}
	return fmt.Sprintf("%s/%s_%d%s",
		path, fileName,
		time.Now().Unix(), fileExt)
}
