package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	if session, err := readAuthSessionFromRequest(r); err == nil {
		data["UserData"] = objx.New(sessionUserData(session))
	}
	t.templ.Execute(w, data)
}

func randomString() (string, error) {
	capacity := 40
	slice := make([]byte, capacity)
	if _, err := rand.Read(slice); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(slice), nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	if err := authCookieConfigError(); err != nil {
		log.Fatal(err)
	}

	GOOGLE_CLIENT_ID := os.Getenv("GOOGLE_CLIENT_ID")
	GOOGLE_CLIENT_SECRET := os.Getenv("GOOGLE_CLIENT_SECRET")
	// FACEBOOK_CLIENT_ID := os.Getenv("FACEBOOK_CLIENT_ID")
	// FACEBOOK_CLIENT_SECRET := os.Getenv("FACEBOOK_CLIENT_SECRET")

	var addr = flag.String("addr", ":8080", "アプリケーションのアドレス")
	flag.Parse() //	フラグを解釈します

	//	Gomniauthのセットアップ
	securityKey, err := randomString()
	if err != nil {
		log.Fatal("authentication security initialization failed")
	}
	gomniauth.SetSecurityKey(securityKey)
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
			Secure:   os.Getenv("APP_ENV") != "development",
			SameSite: http.SameSiteLaxMode,
		})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/upload", MustAuth(&templateHandler{filename: "upload.html"}))
	http.Handle("/uploader", MustAuth(http.HandlerFunc(uploaderHandler)))
	http.Handle("/avatars/",
		http.StripPrefix("/avatars/",
			http.FileServer(http.Dir("./avatars"))))

	//	チャットルームを開始します。
	go r.run()

	//	Webサーバーを起動します
	log.Println("Webサーバーを開始します。ポート：", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
