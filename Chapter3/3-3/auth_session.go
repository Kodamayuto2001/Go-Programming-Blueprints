package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const authSessionLifetime = 8 * time.Hour

type authSession struct {
	UserID    string `json:"userid"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Expires   int64  `json:"exp"`
}

type authSessionContextKey struct{}

func authCookieKey() ([]byte, error) {
	key := []byte(os.Getenv("AUTH_COOKIE_SECRET"))
	if len(key) < 32 {
		return nil, errors.New("AUTH_COOKIE_SECRET must contain at least 32 bytes")
	}
	return key, nil
}

func validSessionUserID(userID string) bool {
	decoded, err := hex.DecodeString(userID)
	return err == nil && len(decoded) == 16
}

func signAuthSession(session authSession) (string, error) {
	key, err := authCookieKey()
	if err != nil {
		return "", err
	}
	if !validSessionUserID(session.UserID) {
		return "", errors.New("invalid authenticated user id")
	}
	session.Expires = time.Now().Add(authSessionLifetime).Unix()
	payload, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(encoded))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + signature, nil
}

func readAuthSession(value string) (authSession, error) {
	var session authSession
	if len(value) == 0 || len(value) > 4096 {
		return session, errors.New("invalid auth cookie")
	}
	key, err := authCookieKey()
	if err != nil {
		return session, err
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return session, errors.New("invalid auth cookie")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return session, errors.New("invalid auth cookie")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return session, errors.New("invalid auth cookie")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(payload, &session) != nil {
		return authSession{}, errors.New("invalid auth cookie")
	}
	if session.Expires <= time.Now().Unix() || !validSessionUserID(session.UserID) {
		return authSession{}, errors.New("expired or invalid auth cookie")
	}
	return session, nil
}

func sessionFromContext(r *http.Request) (authSession, bool) {
	session, ok := r.Context().Value(authSessionContextKey{}).(authSession)
	return session, ok
}

func sessionUserData(session authSession) map[string]interface{} {
	return map[string]interface{}{
		"userid":     session.UserID,
		"name":       session.Name,
		"avatar_url": session.AvatarURL,
	}
}

func readAuthSessionFromRequest(r *http.Request) (authSession, error) {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return authSession{}, err
	}
	return readAuthSession(cookie.Value)
}

func setAuthCookie(w http.ResponseWriter, session authSession) error {
	value, err := signAuthSession(session)
	if err != nil {
		return err
	}
	secure := os.Getenv("APP_ENV") != "development"
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(authSessionLifetime),
		MaxAge:   int(authSessionLifetime.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func authCookieConfigError() error {
	if _, err := authCookieKey(); err != nil {
		return fmt.Errorf("authentication configuration: %w", err)
	}
	return nil
}
