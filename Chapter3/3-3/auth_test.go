package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVerifyAuthCookieRejectsForgedExpiredAndFutureValues(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	now := time.Unix(1_800_000_000, 0)
	token := signAuthCookie("payload", key, now)

	if payload, ok := verifyAuthCookie(token, key, now); !ok || payload != "payload" {
		t.Fatalf("valid token rejected: ok=%v", ok)
	}
	if _, ok := verifyAuthCookie(token+"x", key, now); ok {
		t.Fatal("tampered token accepted")
	}
	if _, ok := verifyAuthCookie(signAuthCookie("payload", key, now.Add(-authCookieMaxAge-time.Second)), key, now); ok {
		t.Fatal("expired token accepted")
	}
	if _, ok := verifyAuthCookie(signAuthCookie("payload", key, now.Add(time.Second)), key, now); ok {
		t.Fatal("future token accepted")
	}
}

func TestMustAuthRejectsNonemptyForgedCookie(t *testing.T) {
	previousKey := authCookieSigningKey
	authCookieSigningKey = []byte("0123456789abcdef0123456789abcdef")
	t.Cleanup(func() { authCookieSigningKey = previousKey })

	called := false
	handler := MustAuth(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	forged := httptest.NewRequest(http.MethodGet, "/private", nil)
	forged.AddCookie(&http.Cookie{Name: "auth", Value: "not-a-signature"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, forged)
	if response.Code != http.StatusTemporaryRedirect || called {
		t.Fatalf("forged cookie reached handler: status=%d called=%v", response.Code, called)
	}
	if response.Header().Get("Location") != "/login" {
		t.Fatal("forged cookie was not redirected to login")
	}
}

func TestMustAuthPassesVerifiedUserDataToHandler(t *testing.T) {
	previousKey := authCookieSigningKey
	authCookieSigningKey = []byte("0123456789abcdef0123456789abcdef")
	t.Cleanup(func() { authCookieSigningKey = previousKey })

	userID := "0123456789abcdef0123456789abcdef"
	payload := base64.StdEncoding.EncodeToString([]byte(`{"userid":"` + userID + `"}`))
	handler := MustAuth(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		got, ok := authUserIDFromRequest(req)
		if !ok || got != userID {
			t.Errorf("verified user id unavailable")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.AddCookie(&http.Cookie{Name: "auth", Value: signAuthCookie(payload, authCookieSigningKey, time.Now())})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusNoContent {
		t.Fatalf("verified cookie handler status = %d, want %d", response.Code, http.StatusNoContent)
	}
}
