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
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

func randomString() string {
	capacity := 40
	slice := make([]byte, capacity)
	rand.Read(slice)
	return base64.URLEncoding.EncodeToString(slice)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
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
	http.Handle("/room", r)
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		//	Value:	クッキーを削除しないブラウザもあるため、Valueの値（ユーザーについてのデータが格納されていた）を空文字列で上書きしている。
		//	MaxAge:	クッキーのMaxAgeの値を-1と指定することで、ブラウザ上のクッキーは即座に削除される。
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/upload", MustAuth(&templateHandler{filename: "upload.html"}))
	http.HandleFunc("/uploader", uploaderHandler)
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
