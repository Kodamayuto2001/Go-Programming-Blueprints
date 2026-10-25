package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const authCookieMaxAge = 12 * time.Hour

var authCookieSigningKey []byte

type authPayloadContextKey struct{}

func newAuthCookieSigningKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func signAuthCookie(payload string, key []byte, issuedAt time.Time) string {
	timestamp := strconv.FormatInt(issuedAt.Unix(), 10)
	message := timestamp + "." + payload
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(message))
	return message + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func verifyAuthCookie(value string, key []byte, now time.Time) (string, bool) {
	if len(key) < 32 {
		return "", false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return "", false
	}
	issuedAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "", false
	}
	issued := time.Unix(issuedAt, 0)
	if issued.After(now) || now.Sub(issued) > authCookieMaxAge {
		return "", false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", false
	}
	return parts[1], true
}

func authPayloadFromRequest(req *http.Request) (string, bool) {
	payload, ok := req.Context().Value(authPayloadContextKey{}).(string)
	return payload, ok && payload != ""
}

func authUserIDFromRequest(req *http.Request) (string, bool) {
	payload, ok := authPayloadFromRequest(req)
	if !ok {
		return "", false
	}
	jsonData, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", false
	}
	var user struct {
		ID string `json:"userid"`
	}
	if err := json.Unmarshal(jsonData, &user); err != nil || !validAvatarUserID(user.ID) {
		return "", false
	}
	return user.ID, true
}

type authHandler struct {
	next http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	cookie, err := req.Cookie("auth")
	if err != nil {
		redirectToLogin(w)
		return
	}
	payload, ok := verifyAuthCookie(cookie.Value, authCookieSigningKey, time.Now())
	if !ok {
		redirectToLogin(w)
		return
	}
	req = req.WithContext(context.WithValue(req.Context(), authPayloadContextKey{}, payload))
	h.next.ServeHTTP(w, req)
}

func redirectToLogin(w http.ResponseWriter) {
	w.Header().Set("Location", "/login")
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func MustAuth(handler http.Handler) http.Handler {
	return &authHandler{next: handler}
}

func validAvatarUserID(userID string) bool {
	if len(userID) != 32 {
		return false
	}
	_, err := hex.DecodeString(userID)
	return err == nil
}
