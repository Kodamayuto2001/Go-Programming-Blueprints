package main

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxUploadBytes int64 = 10 << 20
const maxAvatarDimension = 4096
const maxAvatarStorageBytes int64 = 128 << 20
const maxAvatarFileCount = 512

var uploadSlots = make(chan struct{}, 4)
var avatarPublishMu sync.Mutex

func uploaderHandler(uploadDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		select {
		case uploadSlots <- struct{}{}:
			defer func() { <-uploadSlots }()
		default:
			http.Error(w, "upload capacity reached", http.StatusServiceUnavailable)
			return
		}

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
		content, err := ioutil.ReadAll(file)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				http.Error(w, "upload too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid upload", http.StatusBadRequest)
			return
		}
		userID, ok := authUserIDFromRequest(req)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		imageConfig, imageFormat, err := image.DecodeConfig(bytes.NewReader(content))
		if err != nil || imageConfig.Width < 1 || imageConfig.Height < 1 || imageConfig.Width > maxAvatarDimension || imageConfig.Height > maxAvatarDimension {
			http.Error(w, "invalid image", http.StatusBadRequest)
			return
		}
		if imageFormat != imageFormatForExtension(header.Filename) {
			http.Error(w, "image content does not match its extension", http.StatusBadRequest)
			return
		}

		if err := storeAvatar(uploadDir, userID, header.Filename, content); err != nil {
			if errors.Is(err, errAvatarStorageLimit) {
				http.Error(w, "avatar storage limit reached", http.StatusInsufficientStorage)
				return
			}
			http.Error(w, "upload failed", http.StatusInternalServerError)
			return
		}
		io.WriteString(w, "Upload successful")
	})
}

func imageFormatForExtension(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".png":
		return "png"
	case ".gif":
		return "gif"
	default:
		return ""
	}
}

func createUploadFile(uploadDir, userID, originalFilename string) (*os.File, error) {
	if !validAvatarUserID(userID) {
		return nil, errors.New("invalid user id")
	}
	ext := strings.ToLower(filepath.Ext(originalFilename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
	default:
		return nil, errors.New("unsupported upload extension")
	}
	stagingDir := uploadDir + ".staging"
	if err := os.MkdirAll(stagingDir, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(stagingDir, 0700); err != nil {
		return nil, err
	}
	return ioutil.TempFile(stagingDir, "upload-*")
}

var errAvatarStorageLimit = errors.New("avatar storage limit reached")

const staleUploadMaxAge = time.Hour

func storeAvatar(uploadDir, userID, originalFilename string, content []byte) error {
	if !validAvatarUserID(userID) {
		return errors.New("invalid user id")
	}
	ext := strings.ToLower(filepath.Ext(originalFilename))
	if imageFormatForExtension(originalFilename) == "" {
		return errors.New("unsupported upload extension")
	}
	if err := os.MkdirAll(uploadDir, 0700); err != nil {
		return err
	}

	avatarPublishMu.Lock()
	defer avatarPublishMu.Unlock()

	if err := cleanupStagedUploads(uploadDir+".staging", time.Now().Add(-staleUploadMaxAge)); err != nil {
		return err
	}

	entries, err := ioutil.ReadDir(uploadDir)
	if err != nil {
		return err
	}
	var totalBytes, replacedBytes int64
	var totalFiles, replacedFiles int
	for _, entry := range entries {
		if !entry.Mode().IsRegular() {
			continue
		}
		totalBytes += entry.Size()
		totalFiles++
		if strings.HasPrefix(entry.Name(), userID) {
			replacedBytes += entry.Size()
			replacedFiles++
		}
	}
	if totalBytes-replacedBytes+int64(len(content)) > maxAvatarStorageBytes || totalFiles-replacedFiles+1 > maxAvatarFileCount {
		return errAvatarStorageLimit
	}

	destination, err := createUploadFile(uploadDir, userID, originalFilename)
	if err != nil {
		return err
	}
	temporaryPath := destination.Name()
	if _, err := io.Copy(destination, bytes.NewReader(content)); err != nil {
		_ = destination.Close()
		_ = os.Remove(temporaryPath)
		return err
	}
	if err := destination.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}

	finalName := userID + "-avatar" + ext
	finalPath := filepath.Join(uploadDir, finalName)
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), userID) && entry.Name() != finalName && entry.Mode().IsRegular() {
			if err := os.Remove(filepath.Join(uploadDir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func cleanupStagedUploads(stagingDir string, cutoff time.Time) error {
	entries, err := ioutil.ReadDir(stagingDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "upload-") || !entry.Mode().IsRegular() {
			continue
		}
		if entry.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(stagingDir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
