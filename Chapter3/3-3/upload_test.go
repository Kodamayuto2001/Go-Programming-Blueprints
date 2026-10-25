package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testAvatarUserID = "0123456789abcdef0123456789abcdef"

func validPNG(t *testing.T) []byte {
	t.Helper()
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return imageData.Bytes()
}

func withAuthenticatedTestUser(req *http.Request) *http.Request {
	payload := base64.StdEncoding.EncodeToString([]byte(`{"userid":"` + testAvatarUserID + `"}`))
	return req.WithContext(context.WithValue(req.Context(), authPayloadContextKey{}, payload))
}

func TestCreateUploadFileKeepsUntrustedFilenameInsideUploadDirectory(t *testing.T) {
	tempRoot, err := ioutil.TempDir("", "upload-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempRoot)

	uploadDir := filepath.Join(tempRoot, "avatars")
	file, err := createUploadFile(uploadDir, testAvatarUserID, "../../outside.png")
	if err != nil {
		t.Fatal(err)
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(name)

	if filepath.Dir(filepath.Clean(name)) != filepath.Clean(uploadDir+".staging") {
		t.Fatalf("temporary upload path escaped its staging directory")
	}
	if strings.Contains(filepath.Base(name), "outside") || strings.Contains(filepath.Base(name), "..") || !strings.HasPrefix(filepath.Base(name), "upload-") {
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
		if file, err := createUploadFile(filepath.Join(tempRoot, "avatars"), testAvatarUserID, filename); err == nil {
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
	if _, err := part.Write(validPNG(t)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/uploader", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	uploaderHandler(avatarDir).ServeHTTP(response, withAuthenticatedTestUser(req))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	avatarDirInfo, err := os.Stat(avatarDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := avatarDirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("avatar directory permissions = %04o, want 0700", got)
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
	if !strings.HasPrefix(entries[0].Name(), testAvatarUserID+"-avatar.") {
		t.Fatalf("stored upload lacks the authenticated user prefix")
	}
	if got := entries[0].Mode().Perm(); got != 0600 {
		t.Fatalf("stored upload permissions = %04o, want 0600", got)
	}
}

func TestUploaderHandlerRejectsImageContentWithWrongExtension(t *testing.T) {
	tempRoot, err := ioutil.TempDir("", "upload-test-mismatch-")
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
	if _, err := part.Write(validPNG(t)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/uploader", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	uploaderHandler(filepath.Join(tempRoot, "avatars")).ServeHTTP(response, withAuthenticatedTestUser(req))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if _, err := os.Stat(filepath.Join(tempRoot, "avatars")); !os.IsNotExist(err) {
		t.Fatal("mismatched image upload created a file directory")
	}
}

func TestUploaderHandlerRejectsInvalidImageContent(t *testing.T) {
	tempRoot, err := ioutil.TempDir("", "upload-test-invalid-image-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempRoot)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatarFile", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("not an image")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/uploader", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	uploaderHandler(filepath.Join(tempRoot, "avatars")).ServeHTTP(response, withAuthenticatedTestUser(req))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if _, err := os.Stat(filepath.Join(tempRoot, "avatars")); !os.IsNotExist(err) {
		t.Fatal("invalid image upload created a file directory")
	}
}

func TestStoreAvatarReplacesPreviousFileForUser(t *testing.T) {
	tempRoot, err := ioutil.TempDir("", "upload-test-replace-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempRoot)

	avatarDir := filepath.Join(tempRoot, "avatars")
	content := validPNG(t)
	if err := storeAvatar(avatarDir, testAvatarUserID, "avatar.png", content); err != nil {
		t.Fatal(err)
	}
	if err := storeAvatar(avatarDir, testAvatarUserID, "avatar.png", content); err != nil {
		t.Fatal(err)
	}
	entries, err := ioutil.ReadDir(avatarDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != testAvatarUserID+"-avatar.png" {
		t.Fatalf("replacement left %d files or an unexpected filename", len(entries))
	}
}

func TestAvatarFilesHandlerDoesNotListDirectoryAndSetsSafeHeaders(t *testing.T) {
	tempRoot, err := ioutil.TempDir("", "avatar-files-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempRoot)
	if err := ioutil.WriteFile(filepath.Join(tempRoot, testAvatarUserID+"-avatar.png"), validPNG(t), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(tempRoot, "legacy.txt"), []byte("unexpected public file"), 0600); err != nil {
		t.Fatal(err)
	}

	handler := avatarFilesHandler(tempRoot)
	listing := httptest.NewRecorder()
	handler.ServeHTTP(listing, httptest.NewRequest(http.MethodGet, "/avatars/", nil))
	if listing.Code != http.StatusNotFound {
		t.Fatalf("directory listing status = %d, want %d", listing.Code, http.StatusNotFound)
	}

	imageResponse := httptest.NewRecorder()
	handler.ServeHTTP(imageResponse, httptest.NewRequest(http.MethodGet, "/avatars/"+testAvatarUserID+"-avatar.png", nil))
	if imageResponse.Code != http.StatusOK {
		t.Fatalf("avatar status = %d, want %d", imageResponse.Code, http.StatusOK)
	}
	if got := imageResponse.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := imageResponse.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/png") {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}

	legacyResponse := httptest.NewRecorder()
	handler.ServeHTTP(legacyResponse, httptest.NewRequest(http.MethodGet, "/avatars/legacy.txt", nil))
	if legacyResponse.Code != http.StatusNotFound {
		t.Fatalf("unexpected public file status = %d, want %d", legacyResponse.Code, http.StatusNotFound)
	}
}
