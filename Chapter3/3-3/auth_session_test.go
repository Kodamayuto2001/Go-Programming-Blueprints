package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/common"
	"github.com/stretchr/objx"
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

func TestOAuthStateRequiresSignedValueAndMatchingCookie(t *testing.T) {
	gomniauth.SetSecurityKey("test-only-oauth-signing-key")

	newSignedState := func(nonce string) string {
		value, err := common.NewState("nonce", nonce).SignedBase64(common.GetSecurityKey())
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	requestFor := func(signedState, cookieNonce string) (*http.Request, objx.Map) {
		request := httptest.NewRequest(http.MethodGet, "/auth/callback/google?state="+url.QueryEscape(signedState), nil)
		request.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: cookieNonce})
		query, err := objx.FromURLQuery(request.URL.RawQuery)
		if err != nil {
			t.Fatal(err)
		}
		return request, query
	}

	validRequest, validQuery := requestFor(newSignedState("browser-nonce"), "browser-nonce")
	if err := verifyOAuthState(validRequest, validQuery); err != nil {
		t.Fatalf("expected matching signed state to pass: %v", err)
	}

	mismatchedRequest, mismatchedQuery := requestFor(newSignedState("attacker-nonce"), "browser-nonce")
	if err := verifyOAuthState(mismatchedRequest, mismatchedQuery); err == nil {
		t.Fatal("expected state from a different browser to be rejected")
	}

	tamperedState := newSignedState("browser-nonce") + "tampered"
	tamperedRequest, tamperedQuery := requestFor(tamperedState, "browser-nonce")
	if err := verifyOAuthState(tamperedRequest, tamperedQuery); err == nil {
		t.Fatal("expected tampered state signature to be rejected")
	}
}

func TestOAuthStateCookieHasSecureShortLivedScope(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	response := httptest.NewRecorder()
	setOAuthStateCookie(response, "google", "test-nonce")
	cookie := response.Result().Cookies()[0]
	if cookie.Name != oauthStateCookieName || cookie.Path != "/auth/callback/google" {
		t.Fatalf("unexpected OAuth state cookie scope: %#v", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge <= 0 || cookie.MaxAge > int(oauthStateLifetime.Seconds()) {
		t.Fatalf("unexpected OAuth state cookie security settings: %#v", cookie)
	}
}
