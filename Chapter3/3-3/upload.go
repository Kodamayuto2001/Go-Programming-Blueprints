package main

import (
	"bytes"
	"encoding/hex"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

const maxAvatarSize = 5 << 20

var avatarUploadDirectory = "avatars"

func validAvatarUserID(userID string) bool {
	if len(userID) != 32 {
		return false
	}
	_, err := hex.DecodeString(userID)
	return err == nil
}

func uploaderHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Bound multipart parsing before FormValue/FormFile reads the request body.
	req.Body = http.MaxBytesReader(w, req.Body, maxAvatarSize+(64<<10))
	rawUserID := req.FormValue("userid")
	userID := filepath.Base(rawUserID)
	if rawUserID != userID || !validAvatarUserID(userID) {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	file, _, err := req.FormFile("avatarFile")
	if err != nil {
		http.Error(w, "invalid avatar upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(io.LimitReader(file, maxAvatarSize+1))
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
	filename := filepath.Join(avatarUploadDirectory, userID+extension)
	temp, err := ioutil.TempFile(avatarUploadDirectory, ".avatar-upload-*")
	if err != nil {
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	if err := temp.Close(); err != nil {
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	if err := os.Rename(tempName, filename); err != nil {
		http.Error(w, "avatar storage unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte("成功"))
}
