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
)

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
	t.templ.Execute(w, r)
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
	CLIENT_ID := os.Getenv("CLIENT_ID")
	CLIENT_SECRET := os.Getenv("CLIENT_SECRET")

	var addr = flag.String("addr", ":8080", "アプリケーションのアドレス")
	flag.Parse() //	フラグを解釈します

	//	Gomniauthのセットアップ
	gomniauth.SetSecurityKey(randomString())
	gomniauth.WithProviders(
		google.New(CLIENT_ID, CLIENT_SECRET, "http://localhost:8080/auth/callback/google"),
	)

	r := newRoom()
	// r.tracer = trace.New(os.Stdout)

	// http.Handle("/", &templateHandler{filename: "chat.html"})
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", r)

	//	チャットルームを開始します。
	go r.run()

	//	Webサーバーを起動します
	log.Println("Webサーバーを開始します。ポート：", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
