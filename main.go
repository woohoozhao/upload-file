package main

import (
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const deftPath = "."

func getFileName(header *multipart.FileHeader) string {
	name := filepath.Base(header.Filename)

	name = strings.TrimSuffix(name, filepath.Ext(name))

	if name == "" || name == "." || name == "/" {
		return "default"
	}

	return name
}

const fileExtSize = 512

var mimeExtMap = map[string]string{
	"image/jpeg": ".jpg",
}

func getFileExt(buf []byte) (string, error) {
	if len(buf) < fileExtSize {
		return "", io.EOF
	}
	mimeType := http.DetectContentType(buf)
	ext, ok := mimeExtMap[mimeType]
	if !ok {
		return "", fmt.Errorf("unknown mime type: %s", mimeType)
	}

	return ext, nil
}
func handleUploadStream(res http.ResponseWriter, req *http.Request) {
	mr, err := req.MultipartReader()
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		if part.FormName() != "avatar" {
			err := part.Close()
			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
			continue
		}
	}
	//defer part.Close()
	//ext, err := getFileExt(part)
	slog.Info("stream ext", "ext", ext)
}
func handleUpload(res http.ResponseWriter, req *http.Request) {
	slog.Info("method path", "method", req.Method, "path", req.URL.Path)
	file, header, err := req.FormFile("avatar")
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() {
		err := file.Close()
		if err != nil {
			slog.Error("file close fail", "err", err.Error())
		}
	}()

	fileName := getFileName(header)
	buf := make([]byte, fileExtSize)
	_, err = file.Read(buf)
	file.Seek()
	fileExt, err := getFileExt(buf)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	filePath := setFilePath("", fileName, fileExt)
	slog.Info(filePath)
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		err := dst.Close()
		if err != nil {
			slog.Error("file close has", "err", err.Error())
		}
	}()
	_, err = io.Copy(dst, file)

	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = res.Write([]byte("upload ok"))
	if err != nil {
		slog.Error("res send fail", "err", err.Error())
	}
}

func setFilePath(path, fileName, fileExt string) string {
	if path == "" {
		path = deftPath
	}
	return fmt.Sprintf("%s/%s_%d%s",
		deftPath, fileName,
		time.Now().Unix(), fileExt)
}

func main() {
	//http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/upload", handleUploadStream)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		panic("start server panic")
	}
}
