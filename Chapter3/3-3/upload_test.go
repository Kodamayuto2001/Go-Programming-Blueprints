package main

import (
	"bytes"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateUploadFileKeepsUntrustedFilenameInsideUploadDirectory(t *testing.T) {
	tempRoot, err := ioutil.TempDir("", "upload-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempRoot)

	uploadDir := filepath.Join(tempRoot, "avatars")
	file, err := createUploadFile(uploadDir, "../../outside.png")
	if err != nil {
		t.Fatal(err)
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(name)

	if filepath.Dir(filepath.Clean(name)) != filepath.Clean(uploadDir) {
		t.Fatalf("upload path escaped its directory")
	}
	if strings.Contains(filepath.Base(name), "outside") || strings.Contains(filepath.Base(name), "..") {
		t.Fatalf("client filename influenced the stored path")
	}
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("upload file permissions = %04o, want 0600", got)
	}
}

func TestCreateUploadFileRejectsNonImageExtensions(t *testing.T) {
	tempRoot, err := ioutil.TempDir("", "upload-test-invalid-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempRoot)

	for _, filename := range []string{"avatar.svg", "../../outside", "avatar.exe"} {
		if file, err := createUploadFile(filepath.Join(tempRoot, "avatars"), filename); err == nil {
			name := file.Name()
			_ = file.Close()
			_ = os.Remove(name)
			t.Fatalf("accepted unsupported extension for input %q", filename)
		}
	}
}

func TestUploaderHandlerRejectsOversizedRequest(t *testing.T) {
	tempRoot, err := ioutil.TempDir("", "upload-test-size-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempRoot)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatarFile", "avatar.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(make([]byte, int(maxUploadBytes+1))); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/uploader", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	uploaderHandler(filepath.Join(tempRoot, "avatars")).ServeHTTP(response, req)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if _, err := os.Stat(filepath.Join(tempRoot, "avatars")); !os.IsNotExist(err) {
		t.Fatalf("oversized request created upload directory: stat error = %v", err)
	}
}

func TestUploaderHandlerStoresUploadInAvatarDirectory(t *testing.T) {
	tempRoot, err := ioutil.TempDir("", "upload-test-avatar-dir-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempRoot)
	avatarDir := filepath.Join(tempRoot, "avatars")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatarFile", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("test image bytes")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/uploader", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	uploaderHandler(avatarDir).ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	entries, err := ioutil.ReadDir(avatarDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("avatar directory contains %d files, want 1", len(entries))
	}
	if filepath.Ext(entries[0].Name()) != ".png" {
		t.Fatalf("stored upload has extension %q, want .png", filepath.Ext(entries[0].Name()))
	}
	if got := entries[0].Mode().Perm(); got != 0600 {
		t.Fatalf("stored upload permissions = %04o, want 0600", got)
	}
}
