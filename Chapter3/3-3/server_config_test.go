package main

import "testing"

func TestValidateServerConfigAllowsLoopbackDevelopment(t *testing.T) {
	if err := validateServerConfig("127.0.0.1:8080", "", "", "http://localhost:8080/auth/callback/google"); err != nil {
		t.Fatalf("loopback development config rejected: %v", err)
	}
}

func TestValidateServerConfigRejectsPublicHTTPAndMismatchedTLS(t *testing.T) {
	if err := validateServerConfig(":8080", "", "", "http://localhost:8080/auth/callback/google"); err == nil {
		t.Fatal("public HTTP binding accepted")
	}
	if err := validateServerConfig("127.0.0.1:8080", "cert.pem", "", "http://localhost:8080/auth/callback/google"); err == nil {
		t.Fatal("partial TLS configuration accepted")
	}
	if err := validateServerConfig("127.0.0.1:8080", "", "", "http://example.com/auth/callback/google"); err == nil {
		t.Fatal("non-loopback HTTP OAuth callback accepted")
	}
	if err := validateServerConfig("127.0.0.1:8081", "", "", "http://localhost:8080/auth/callback/google"); err == nil {
		t.Fatal("loopback HTTP OAuth callback on a different port accepted")
	}
	if err := validateServerConfig("127.0.0.1:8443", "cert.pem", "key.pem", "http://localhost:8080/auth/callback/google"); err == nil {
		t.Fatal("HTTPS deployment accepted an HTTP callback")
	}
}

func TestValidateServerConfigAllowsHTTPSDeployment(t *testing.T) {
	if err := validateServerConfig(":8443", "cert.pem", "key.pem", "https://chat.example.com/auth/callback/google"); err != nil {
		t.Fatalf("HTTPS deployment config rejected: %v", err)
	}
}

func TestValidateServerConfigRejectsMalformedOAuthCallback(t *testing.T) {
	for _, callback := range []string{
		"/auth/callback/google",
		"http://localhost:8080/auth/callback/google?next=https://attacker.example",
		"http://localhost/auth/callback/google",
		"http://user@localhost:8080/auth/callback/google",
		"ftp://localhost:8080/auth/callback/google",
	} {
		if err := validateServerConfig("127.0.0.1:8080", "", "", callback); err == nil {
			t.Fatalf("malformed OAuth callback accepted: %q", callback)
		}
	}
}
