package main

import (
	"io/ioutil"
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

	uploadDir := filepath.Join(tempRoot, "uploads")
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
		if file, err := createUploadFile(filepath.Join(tempRoot, "uploads"), filename); err == nil {
			name := file.Name()
			_ = file.Close()
			_ = os.Remove(name)
			t.Fatalf("accepted unsupported extension for input %q", filename)
		}
	}
}
