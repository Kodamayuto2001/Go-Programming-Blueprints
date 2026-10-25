package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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

const oauthStateCookieName = "oauth_state"
const oauthStateMaxAge = 5 * time.Minute

func newOAuthState() (string, error) {
	state := make([]byte, 32)
	if _, err := rand.Read(state); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(state), nil
}

func oauthCallbackStateMatches(req *http.Request) bool {
	cookie, err := req.Cookie(oauthStateCookieName)
	if err != nil || len(cookie.Value) < 40 {
		return false
	}
	state, err := gomniauth.StateFromParam(req.URL.Query().Get("state"))
	if err != nil {
		return false
	}
	csrf := state.Get("csrf").Str()
	return len(csrf) == len(cookie.Value) && subtle.ConstantTimeCompare([]byte(csrf), []byte(cookie.Value)) == 1
}

func clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/auth/callback/google",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (u chatUser) UniqueID() string {
	return u.uniqueID
}

// loginHandlerはサードパーティーへのログインの処理を受け持ちます。
// パスの形式： /auth/{action}/{provider}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	segs := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(segs) != 3 || segs[0] != "auth" {
		http.NotFound(w, r)
		return
	}
	action := segs[1]
	providerName := segs[2]
	provider, err := gomniauth.Provider(providerName)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch action {
	case "login":
		csrfState, err := newOAuthState()
		if err != nil {
			http.Error(w, "authentication could not be started", http.StatusInternalServerError)
			return
		}
		loginURL, err := provider.GetBeginAuthURL(gomniauth.NewState("csrf", csrfState), nil)
		if err != nil {
			http.Error(w, "authentication could not be started", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     oauthStateCookieName,
			Value:    csrfState,
			Path:     "/auth/callback/" + providerName,
			MaxAge:   int(oauthStateMaxAge.Seconds()),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
	case "callback":
		if !oauthCallbackStateMatches(r) {
			clearOAuthStateCookie(w)
			http.Error(w, "invalid authentication state", http.StatusBadRequest)
			return
		}
		clearOAuthStateCookie(w)

		query, err := objx.FromURLQuery(r.URL.RawQuery)
		if err != nil {
			http.Error(w, "invalid authentication response", http.StatusBadRequest)
			return
		}
		creds, err := provider.CompleteAuth(query)
		if err != nil {
			http.Error(w, "authentication could not be completed", http.StatusBadGateway)
			return
		}

		user, err := provider.GetUser(creds)
		if err != nil {
			http.Error(w, "user information could not be loaded", http.StatusBadGateway)
			return
		}

		// m := md5.New()
		// io.WriteString(m, strings.ToLower(user.Email()))
		// userID := fmt.Sprintf("%x", m.Sum(nil))

		chatUser := &chatUser{User: user}
		m := md5.New()
		io.WriteString(m, strings.ToLower(user.Email()))
		chatUser.uniqueID = fmt.Sprintf("%x", m.Sum(nil))
		avatarURL, err := avatars.GetAvatarURL(chatUser)
		if err != nil {
			http.Error(w, "avatar information could not be loaded", http.StatusBadGateway)
			return
		}
		//	データを保存します
		// authCookieValue := objx.New(map[string]interface{}{
		// 	"userid":     userID,
		// 	"name":       user.Name(),
		// 	"avatar_url": user.AvatarURL(),
		// 	"email":      user.Email(),
		// }).MustBase64()

		authCookiePayload := objx.New(map[string]interface{}{
			"userid":     chatUser.uniqueID,
			"name":       user.Name(),
			"avatar_url": avatarURL,
		}).MustBase64()
		authCookieValue := signAuthCookie(authCookiePayload, authCookieSigningKey, time.Now())

		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    authCookieValue,
			Path:     "/",
			MaxAge:   int(authCookieMaxAge.Seconds()),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	default:
		http.NotFound(w, r)
	}
}
