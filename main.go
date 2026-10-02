package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const deftPath = "."

func getFileName(filename string) string {
	name := filepath.Base(filename)

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
			_, _ = res.Write([]byte("upload ok"))
			return
		}
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		if part.FormName() != "avatar" {
			_ = part.Close()
			continue
		}

		head := make([]byte, fileExtSize)
		n, err := part.Read(head)
		if err != nil && err != io.EOF {
			_ = part.Close()
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		fileExt, err := getFileExt(head[:n])
		if err != nil {
			_ = part.Close()
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		fileName := getFileName(part.FileName())

		filePath := setFilePath("", fileName, fileExt)

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = dst.Write(head[:n])
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(dst, part); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = res.Write([]byte("upload ok"))
		if err != nil {
			slog.Error("res send fail", "err", err.Error())
		}
		_ = dst.Close()
		_ = part.Close()
		return
	}
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

	fileName := getFileName(header.Filename)
	buf := make([]byte, fileExtSize)
	n, err := file.Read(buf)
	if err != nil {
		slog.Error("file read fail", "err", err.Error())
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	fileExt, err := getFileExt(buf[:n])
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		slog.Error("file seek fail", "err", err.Error())
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
		slog.Error("http listen err", "err", err.Error())
		panic("start server panic")
	}
}
