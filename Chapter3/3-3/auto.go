package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stretchr/gomniauth"
	"github.com/stretchr/objx"

	gomniauthcommon "github.com/stretchr/gomniauth/common"
)

type ChatUser interface {
	UniqueID() string
	AvatarURL() string
}

// 型の埋め込み(type embedding)
// gomniauth/common.User型
// chatUserは自動的にGomniauthのUserインターフェースを実装
// AvatarURLはGomniauthのUserインターフェースでもAvatarURLメソッドが定義されている
type chatUser struct {
	gomniauthcommon.User
	uniqueID string
}

func (u chatUser) UniqueID() string {
	return u.uniqueID
}

type authHandler struct {
	next http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth")
	if err != nil {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
	session, err := readAuthSession(cookie.Value)
	if err != nil {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
	ctx := context.WithValue(r.Context(), authSessionContextKey{}, session)
	h.next.ServeHTTP(w, r.WithContext(ctx))
}

func MustAuth(handler http.Handler) http.Handler {
	return &authHandler{next: handler}
}

const (
	oauthStateCookieName = "oauth_state"
	oauthStateLifetime   = 10 * time.Minute
)

func newOAuthState() (*gomniauthcommon.State, string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", err
	}
	nonceValue := base64.RawURLEncoding.EncodeToString(nonce)
	return gomniauthcommon.NewState("nonce", nonceValue), nonceValue, nil
}

func verifyOAuthState(r *http.Request, query objx.Map) error {
	cookie, err := r.Cookie(oauthStateCookieName)
	if err != nil {
		return errors.New("missing OAuth state cookie")
	}
	signedState := query.Get("state").Str()
	if signedState == "" {
		return errors.New("missing OAuth state")
	}
	state, err := objx.FromSignedBase64(signedState, gomniauthcommon.GetSecurityKey())
	if err != nil {
		return errors.New("invalid OAuth state")
	}
	stateNonce := state.Get("nonce").Str()
	if stateNonce == "" || !hmac.Equal([]byte(cookie.Value), []byte(stateNonce)) {
		return errors.New("OAuth state does not match this browser")
	}
	return nil
}

func setOAuthStateCookie(w http.ResponseWriter, providerName, nonce string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    nonce,
		Path:     "/auth/callback/" + providerName,
		Expires:  time.Now().Add(oauthStateLifetime),
		MaxAge:   int(oauthStateLifetime.Seconds()),
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") != "development",
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthStateCookie(w http.ResponseWriter, providerName string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/auth/callback/" + providerName,
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") != "development",
		SameSite: http.SameSiteLaxMode,
	})
}

// loginHandlerはサードパーティーへのログインの処理を受け持ちます。
// パスの形式： /auth/{action}/{provider}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	segs := strings.Split(r.URL.Path, "/")
	if len(segs) != 4 || segs[1] != "auth" || segs[2] == "" || segs[3] == "" {
		http.NotFound(w, r)
		return
	}
	action := segs[2]
	providerName := segs[3]
	provider, err := gomniauth.Provider(providerName)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch action {
	case "login":
		state, nonce, err := newOAuthState()
		if err != nil {
			http.Error(w, "authentication is unavailable", http.StatusServiceUnavailable)
			return
		}
		loginUrl, err := provider.GetBeginAuthURL(state, nil)
		if err != nil {
			http.Error(w, "authentication provider unavailable", http.StatusServiceUnavailable)
			return
		}
		setOAuthStateCookie(w, providerName, nonce)
		w.Header().Set("Location", loginUrl)
		w.WriteHeader(http.StatusTemporaryRedirect)
	case "callback":
		defer clearOAuthStateCookie(w, providerName)
		query, err := objx.FromURLQuery(r.URL.RawQuery)
		if err != nil {
			http.Error(w, "invalid authentication response", http.StatusBadRequest)
			return
		}
		if err := verifyOAuthState(r, query); err != nil {
			http.Error(w, "invalid authentication response", http.StatusBadRequest)
			return
		}
		creds, err := provider.CompleteAuth(query)
		if err != nil {
			http.Error(w, "authentication could not be completed", http.StatusUnauthorized)
			return
		}

		user, err := provider.GetUser(creds)
		if err != nil {
			http.Error(w, "user profile unavailable", http.StatusBadGateway)
			return
		}

		chatUser := &chatUser{User: user}
		userID := sha256.Sum256([]byte(strings.ToLower(user.Email())))
		chatUser.uniqueID = hex.EncodeToString(userID[:])
		avatarURL, err := avatars.GetAvatarURL(chatUser)
		if err != nil {
			http.Error(w, "user avatar unavailable", http.StatusBadGateway)
			return
		}
		if err := setAuthCookie(w, authSession{
			UserID:    chatUser.uniqueID,
			Name:      user.Name(),
			AvatarURL: avatarURL,
		}); err != nil {
			http.Error(w, "authentication is not configured", http.StatusServiceUnavailable)
			return
		}
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	default:
		http.NotFound(w, r)
	}
}
