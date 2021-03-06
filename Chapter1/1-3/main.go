package main

import (
	"log"
	"net/http"
	"text/template"
	"path/filepath"
	"sync"
	"flag"
)

type templateHandler struct {
	once		sync.Once
	filename	string
	templ		*template.Template
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(
			template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	t.templ.Execute(w, r)
}

// func main() {
// 	r := newRoom()
// 	http.Handle("/", &templateHandler{filename: "chat.html"})
// 	http.Handle("/room", r)
// 	//	チャットルームを開始します。
// 	go r.run()
// 	//	Webサーバーを起動します。
// 	if err := http.ListenAndServe(":8080", nil); err != nil {
// 		log.Fatal("ListenAndServe:", err)
// 	}
// }

func main() {
	var addr = flag.String("addr", ":8080", "アプリケーションのアドレス")
	flag.Parse()	//	フラグを解釈します
	r := newRoom()
	http.Handle("/", &templateHandler{filename: "chat.html"})
	http.Handle("/room", r)
	//	チャットルームを開始します。
	go r.run()
	//	Webサーバーを起動します
	log.Println("Webサーバーを開始します。ポート：", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}