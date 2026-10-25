package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/joho/godotenv"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/objx"
)

// var avatars Avatar = UseFileSystemAvatar
var avatars Avatar = TryAvatars{
	UseFileSystemAvatar,
	UseAuthAvatar,
	UseGravatar}

type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(
			template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	data := map[string]interface{}{
		"Host": r.Host,
	}
	if payload, ok := authPayloadFromRequest(r); ok {
		data["UserData"] = objx.MustFromBase64(payload)
	}
	t.templ.Execute(w, data)
}

func randomString() string {
	capacity := 40
	slice := make([]byte, capacity)
	if _, err := rand.Read(slice); err != nil {
		log.Fatal("安全な乱数を取得できませんでした：", err)
	}
	return base64.URLEncoding.EncodeToString(slice)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	authCookieSigningKey, err = newAuthCookieSigningKey()
	if err != nil {
		log.Fatal("認証Cookie用の鍵を生成できませんでした：", err)
	}

	GOOGLE_CLIENT_ID := os.Getenv("GOOGLE_CLIENT_ID")
	GOOGLE_CLIENT_SECRET := os.Getenv("GOOGLE_CLIENT_SECRET")
	// FACEBOOK_CLIENT_ID := os.Getenv("FACEBOOK_CLIENT_ID")
	// FACEBOOK_CLIENT_SECRET := os.Getenv("FACEBOOK_CLIENT_SECRET")

	var addr = flag.String("addr", ":8080", "アプリケーションのアドレス")
	flag.Parse() //	フラグを解釈します

	//	Gomniauthのセットアップ
	gomniauth.SetSecurityKey(randomString())
	gomniauth.WithProviders(
		google.New(GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, "http://localhost:8080/auth/callback/google"),
	)

	r := newRoom()
	// r := newRoom(UseAuthAvatar)
	// r := newRoom(UseGravatar)
	// r := newRoom(UseFileSystemAvatar)
	// r.tracer = trace.New(os.Stdout)

	// http.Handle("/", &templateHandler{filename: "chat.html"})
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", MustAuth(r))
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		//	Value:	クッキーを削除しないブラウザもあるため、Valueの値（ユーザーについてのデータが格納されていた）を空文字列で上書きしている。
		//	MaxAge:	クッキーのMaxAgeの値を-1と指定することで、ブラウザ上のクッキーは即座に削除される。
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/upload", MustAuth(&templateHandler{filename: "upload.html"}))
	http.Handle("/uploader", MustAuth(uploaderHandler("./avatars")))
	http.Handle("/avatars/", avatarFilesHandler("./avatars"))

	//	チャットルームを開始します。
	go r.run()

	//	Webサーバーを起動します
	log.Println("Webサーバーを開始します。ポート：", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func avatarFilesHandler(avatarDir string) http.Handler {
	avatarFiles := http.StripPrefix("/avatars/", http.FileServer(http.Dir(avatarDir)))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if contentType := mime.TypeByExtension(filepath.Ext(req.URL.Path)); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		if strings.HasSuffix(req.URL.Path, "/") || filepath.Base(req.URL.Path) == "." {
			http.NotFound(w, req)
			return
		}
		avatarFiles.ServeHTTP(w, req)
	})
}
