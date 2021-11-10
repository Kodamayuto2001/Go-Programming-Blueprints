package main

import (
	"log"
	"net/http"
)

func main() {
	//	net/httpパッケージを利用し、ルートのパスつまり/へのリクエストを待ち受ける
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//	リクエストを受け取ると、ハードコードされたHTMLを返す
		w.Write([]byte(`
			<html>
				<head>
					<title>チャット</title>
				</head>
				<body>
					チャットしましょう！
				</body>
			</html>
		`))
	})

	//	ListenAndServeメソッドを使い、ポート3000上でWebサーバーを開始する。
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}