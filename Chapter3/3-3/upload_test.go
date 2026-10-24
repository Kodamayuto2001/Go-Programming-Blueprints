package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const testAvatarUserID = "0123456789abcdef0123456789abcdef"

func pngAvatar(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func avatarRequest(t *testing.T, userID, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("userid", userID); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("avatarFile", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/uploader", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func withAvatarDirectory(t *testing.T) string {
	t.Helper()
	previous := avatarUploadDirectory
	avatarUploadDirectory = t.TempDir()
	t.Cleanup(func() { avatarUploadDirectory = previous })
	return avatarUploadDirectory
}

func TestUploaderRejectsPathTraversalUserID(t *testing.T) {
	dir := withAvatarDirectory(t)
	rec := httptest.NewRecorder()
	uploaderHandler(rec, avatarRequest(t, "../../outside", "avatar.png", pngAvatar(t)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected files created: %d", len(entries))
	}
}

func TestUploaderIgnoresClientFilenameAndStoresValidatedPNG(t *testing.T) {
	dir := withAvatarDirectory(t)
	rec := httptest.NewRecorder()
	uploaderHandler(rec, avatarRequest(t, testAvatarUserID, "../../outside.html", pngAvatar(t)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, testAvatarUserID+".png")); err != nil {
		t.Fatalf("validated avatar was not stored under the server-generated name: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != testAvatarUserID+".png" {
		t.Fatalf("unexpected files stored: %#v", entries)
	}
}

func TestUploaderRejectsNonImageContent(t *testing.T) {
	dir := withAvatarDirectory(t)
	rec := httptest.NewRecorder()
	uploaderHandler(rec, avatarRequest(t, testAvatarUserID, "avatar.png", []byte("not an image")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected files created: %d", len(entries))
	}
}

func TestUploaderRouteRequiresAuthCookie(t *testing.T) {
	dir := withAvatarDirectory(t)
	rec := httptest.NewRecorder()
	handler := MustAuth(http.HandlerFunc(uploaderHandler))
	handler.ServeHTTP(rec, avatarRequest(t, testAvatarUserID, "avatar.png", pngAvatar(t)))
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTemporaryRedirect)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unexpected files created before authentication: %d", len(entries))
	}
}
