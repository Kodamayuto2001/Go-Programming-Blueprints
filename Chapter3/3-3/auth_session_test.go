package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testAuthCookieSecret = "test-only-secret-with-at-least-32-bytes"
const testAuthUserID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestAuthSessionRequiresStrongSecretAndValidUserID(t *testing.T) {
	t.Setenv("AUTH_COOKIE_SECRET", "short")
	if _, err := signAuthSession(authSession{UserID: testAuthUserID}); err == nil {
		t.Fatal("expected short secret to be rejected")
	}
	t.Setenv("AUTH_COOKIE_SECRET", testAuthCookieSecret)
	if _, err := signAuthSession(authSession{UserID: "../not-a-user"}); err == nil {
		t.Fatal("expected invalid user id to be rejected")
	}
}

func TestAuthSessionSignatureRejectsTampering(t *testing.T) {
	t.Setenv("AUTH_COOKIE_SECRET", testAuthCookieSecret)
	value, err := signAuthSession(authSession{UserID: testAuthUserID, Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		t.Fatal("signed cookie did not have payload and signature")
	}
	tampered := parts[0] + "." + strings.Repeat("A", len(parts[1]))
	if _, err := readAuthSession(tampered); err == nil {
		t.Fatal("expected modified signature to be rejected")
	}
	session, err := readAuthSession(value)
	if err != nil {
		t.Fatal(err)
	}
	if session.UserID != testAuthUserID || session.Name != "test" {
		t.Fatal("signed session did not round-trip")
	}
	if _, err := readAuthSession(strings.Repeat("A", 4097)); err == nil {
		t.Fatal("expected oversized cookie to be rejected")
	}
}

func TestAuthCookieHasSecureDefaults(t *testing.T) {
	t.Setenv("AUTH_COOKIE_SECRET", testAuthCookieSecret)
	t.Setenv("APP_ENV", "production")
	response := httptest.NewRecorder()
	if err := setAuthCookie(response, authSession{UserID: testAuthUserID}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	cookie := response.Result().Cookies()[0]
	request.AddCookie(cookie)
	session, err := readAuthSessionFromRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if session.UserID != testAuthUserID {
		t.Fatal("request did not decode the signed session")
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge <= 0 {
		t.Fatalf("unexpected auth cookie security settings: %#v", cookie)
	}
}
