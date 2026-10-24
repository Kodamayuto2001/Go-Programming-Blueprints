package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

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
		loginUrl, err := provider.GetBeginAuthURL(nil, nil)
		if err != nil {
			http.Error(w, "authentication provider unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Location", loginUrl)
		w.WriteHeader(http.StatusTemporaryRedirect)
	case "callback":
		query, err := objx.FromURLQuery(r.URL.RawQuery)
		if err != nil {
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
