package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"
)

const fileExtSize = 512

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
			_ = part.Close()
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
			http.Error(res, err.Error(), http.StatusUnsupportedMediaType)
			return
		}

		fileName := getFileName(part.FileName())

		filePath := setFilePath("", fileName, fileExt)

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			_ = part.Close()
			if dst != nil {
				_ = dst.Close()
			}
			return
		}
		_, err = dst.Write(head[:n])
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			_ = dst.Close()
			_ = part.Close()
			return
		}

		if _, err := io.Copy(dst, part); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			_ = dst.Close()
			_ = part.Close()
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

func main() {
	if os.MkdirAll(deftPath, 0777) != nil {
		slog.Error("mkdir fail")
		panic("start server panic")
	}
	//http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/upload", handleUploadStream)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		slog.Error("http listen err", "err", err)
		panic("start server panic")
	}

}
