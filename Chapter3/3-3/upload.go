package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
)

const maxAvatarSize = 5 << 20

var avatarUploadDirectory = "avatars"

func uploaderHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, ok := sessionFromContext(req); !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	// Bound multipart parsing before FormFile reads the request body.
	req.Body = http.MaxBytesReader(w, req.Body, maxAvatarSize+(64<<10))

	file, _, err := req.FormFile("avatarFile")
	if err != nil {
		http.Error(w, "invalid avatar upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarSize+1))
	if err != nil || len(data) > maxAvatarSize {
		http.Error(w, "avatar is too large or unreadable", http.StatusBadRequest)
		return
	}

	contentType := http.DetectContentType(data)
	var extension string
	switch contentType {
	case "image/jpeg":
		extension = ".jpg"
	case "image/png":
		extension = ".png"
	default:
		http.Error(w, "only JPEG and PNG avatars are accepted", http.StatusBadRequest)
		return
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 {
		http.Error(w, "invalid or oversized image", http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(avatarUploadDirectory, 0700); err != nil {
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	session, ok := sessionFromContext(req)
	if !ok || !validSessionUserID(session.UserID) {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	userIDBytes, err := hex.DecodeString(session.UserID)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	filename := hex.EncodeToString(userIDBytes) + extension
	root, err := os.OpenRoot(avatarUploadDirectory)
	if err != nil {
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	defer root.Close()
	randomName := make([]byte, 16)
	if _, err := rand.Read(randomName); err != nil {
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	tempName := ".avatar-upload-" + hex.EncodeToString(randomName)
	temp, err := root.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	defer root.Remove(tempName)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	if err := temp.Close(); err != nil {
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	if err := root.Rename(tempName, filename); err != nil {
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte("成功"))
}
