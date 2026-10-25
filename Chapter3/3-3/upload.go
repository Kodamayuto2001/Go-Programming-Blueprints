package main

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxUploadBytes int64 = 10 << 20

func uploaderHandler(uploadDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(w, req.Body, maxUploadBytes)
		file, header, err := req.FormFile("avatarFile")
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				http.Error(w, "upload too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid upload", http.StatusBadRequest)
			return
		}
		defer file.Close()

		destination, err := createUploadFile(uploadDir, header.Filename)
		if err != nil {
			http.Error(w, "unsupported upload", http.StatusBadRequest)
			return
		}
		if _, err := io.Copy(destination, file); err != nil {
			_ = destination.Close()
			_ = os.Remove(destination.Name())
			http.Error(w, "upload failed", http.StatusInternalServerError)
			return
		}
		if err := destination.Close(); err != nil {
			_ = os.Remove(destination.Name())
			http.Error(w, "upload failed", http.StatusInternalServerError)
			return
		}
		io.WriteString(w, "Upload successful")
	})
}

func createUploadFile(uploadDir, originalFilename string) (*os.File, error) {
	ext := strings.ToLower(filepath.Ext(originalFilename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
	default:
		return nil, errors.New("unsupported upload extension")
	}
	if err := os.MkdirAll(uploadDir, 0700); err != nil {
		return nil, err
	}
	return ioutil.TempFile(uploadDir, "avatar-*"+ext)
}
