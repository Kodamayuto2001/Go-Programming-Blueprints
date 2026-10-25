package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/gomniauth"
)

func TestOAuthCallbackStateRequiresSignedStateBoundToCookie(t *testing.T) {
	gomniauth.SetSecurityKey("test-only-state-signing-key")
	csrf, err := newOAuthState()
	if err != nil {
		t.Fatalf("could not create OAuth state: %v", err)
	}
	signedState, err := gomniauth.NewState("csrf", csrf).SignedBase64(gomniauth.GetSecurityKey())
	if err != nil {
		t.Fatalf("could not sign OAuth state: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/callback/google?state="+url.QueryEscape(signedState), nil)
	if oauthCallbackStateMatches(req) {
		t.Fatal("OAuth callback without a browser state cookie was accepted")
	}
	req.Header.Set("Cookie", oauthStateCookieName+"=different-state")
	if oauthCallbackStateMatches(req) {
		t.Fatal("OAuth callback with a mismatched state cookie was accepted")
	}
	req.Header.Set("Cookie", oauthStateCookieName+"="+csrf)
	if !oauthCallbackStateMatches(req) {
		t.Fatal("valid browser-bound OAuth state was rejected")
	}
}

func TestLoginHandlerRejectsMalformedPathAndNonGetMethod(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
		want   int
	}{
		{method: http.MethodGet, path: "/auth/", want: http.StatusNotFound},
		{method: http.MethodPost, path: "/auth/login/google", want: http.StatusMethodNotAllowed},
	} {
		req := httptest.NewRequest(test.method, test.path, nil)
		response := httptest.NewRecorder()
		loginHandler(response, req)
		if response.Code != test.want {
			t.Errorf("%s %s status=%d, want %d", test.method, test.path, response.Code, test.want)
		}
	}
}
