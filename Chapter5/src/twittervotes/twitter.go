package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joeshaw/envdecode"
)

/*
	dial関数はまず、接続を表すconnが閉じられているかどうかを確認します。
	そして新しい接続を開き、connの値を更新します。
	接続が異常終了したり(TwitterのAPIでは時々このようなことが起こります)
	我々が意図的に接続を閉じたりした場合には、再接続が試みられます。
	この際に、接続のゾンビ化について心配する必要はありません。
	データベースから取得した選択しに関する最新のデータを反映させるために、定期的に接続を閉じて接続しなおすことにします。
	接続を閉じるための関数を呼び出すだけではなく、レスポンスの本体を読み込むのに使われるio.ReadCloserも閉じる必要があります。
*/

var conn net.conn

func dial(netw, addr string) (net.Conn, error) {
	if conn != nil {
		conn.Close()
		conn = nil
	}

	netc, err := net.DialTimeout(netw, addr, 5*time.Second)
	if err != nil {
		return nil, err
	}

	conn = netc
	return netc, nil
}

/*
	closeConnを呼び出せば、いつでもTwitterとの現在の接続を切断してクリーンアップを行えるようになります。
	ほとんどの場合、直後にデータベースから新しい選択しのリストを取得して接続が再び開かれることになります。
	しかし、プログラムの終了時（Ctrl + Cが押された場合）には、最後にcloseConnの処理だけが実行されます。
*/

var reader io.ReadCloser

func closeConn() {
	if conn != nil {
		conn.Close()
	}

	if reader != nil {
		reader.Close()
	}
}

/*
	ここでは、環境変数の値を格納するための構造体が定義されています。
	これらの値は、Twitterとの認証に使われます。
	この構造体はここでしか使わないので、インラインで定義して匿名型のtsという変数を用意しています。
	var ts struct...という変わったコードが使われているのは、このためです。
	そして、Joe Shawが作成したエレガントなenvdecodeパッケージを使い、環境変数の値を読み込みます。
	このパッケージを利用するには、go get github.com/joeshaw/envdecodeを実行しておく必要があります。
	指定されたフィールドの値がすべて読み込まれるとともに、required（必須）と指定されたフィールドの値を取得できなかった場合にはエラーが返されます。
	このエラーをlog.Fatallnで出力することで、Twitterの認証がないとこのプログラムは動かないということを知らせています。
	そのためlogパッケージもインポートしておく必要があります。
	構造体の各フィールドの中で、バッククォートに囲まれている部分はタグと呼ばれます。
	リフレクションのAPIを使うとこのタグにアクセスできます。
	envdecodeはこの仕組みを使って、環境変数の名前を取得しています。
	Tyler Bunnellと筆者はこのパッケージを修正してrequired引数を加え、値が空あるいは存在しない環境変数があった場合に、エラーが返されるようにしました。
	必要な値が用意出来たら、これらをもとにoauth.Credentialsとoauth.Clientというオブジェクトを生成し、Twitterに対してのリクエストの認証を行います。
	これらはGary Burdによるgo-oauthパッケージに含まれています
*/

var (
	authClient *oauth.Client
	creds      *oauth.Credentials
)

func setupTwitterAuth() {
	var ts struct {
		ConsumerKey    string `env:"SP_TWITTER_KEY,required"`
		ConsumerSecret string `env:"SP_TWITTER_SECRET,required"`
		AccessToken    string `env:"SP_TWITTER_ACCESSTOKEN,required"`
		AccessSecret   string `env:"SP_TWITTER_ACCESSSECRET,required"`
	}

	if err := envdecode.Decode(&ts); err != nil {
		log.Fatalln(err)
	}

	creds = &oauth.Credentials{
		Token:  ts.AccessToken,
		Secret: ts.AccessSecret,
	}

	authClient = &oauth.Client{
		Credentials: oauth.Credentials{
			Token:  ts.ConsumerKey,
			Secret: ts.ConsumerSecret,
		},
	}
}

var (
	authSetupOnce sync.Once
	httpClient    *http.Client
)

func makeRequest(req *http.Request, params url.Values) (*http.Response, error) {
	authSetupOnce.Do(func() {
		setupTwitterAuth()
		httpClient = &http.Client{
			Transport: &http.Transport{
				Dial: dial,
			},
		}
	})

	formEnc := params.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(formEnc)))
	req.Header.Set("Authorization", authClient.AuthorizationHeader(creds, "POST", req.URL, params))
	return httpClient.Do(req)
}

/*
	ここでは、まずloadOptions関数を呼び出してすべての投票での選択しを取得しています。次にurl.Parseを使い、Twitter側のエンドポイントを指すurl.URLオブジェクトを生成しています。queryというurl.Valuesオブジェクトも生成し、選択肢のリストをカンマ区切りの文字列として指定しています。TwitterのAPIでは、url.ValuesオブジェクトがエンコードされたものをPOSTリクエストとして送信する必要があります。そのリクエストを表す*http.Requestをhttp.NewRequestで作成し、queryオブジェクトとともにmakeRequestに渡します。リクエストに成功すると、レスポンスの本体をもとにjson.Decoderを生成し、無限ループの中でDecodeメソッドを呼び出してデータを読み込みます。(主に接続が閉じられたなどの理由で)エラーが発生したら、ループから抜け出して呼び出し元に戻ります。読み込むツイートが存在する場合には、デコードされたツイートがtweet変数にセットされます。そしてこの中のTextプロパティに、140文字のツイート本文がセットされています。全ての選択しについて、ツイートの中で言及されている場合にはvotesチャネルにその選択肢を送信するという処理が行われます。つまり、1つのツイートの中で複数の選択しに対して投票するということが可能です。
	投票の種類によっては、このルールを変更するべきかもしれません。

		※votesチャネルには chan<- stringという型が指定されており、送信専用です。ここからデータを受け取ることはできません。<-はメッセージの流れる向きを表す矢印のようなものであり、逆向き（受信専用）に指定することもできます（<-chan string）。このような矢印も、コードの意図を表すうえで大きな役割を果たしています。readFromTwitter関数ではチャネルから投票のデータを受信することはなく、送信するだけだということを明示できます。

	Decodeがエラーを返すたびにプログラムを終了するというのは、頑健なやり方とは言えません。TwitterのAPIドキュメントによると、接続は切断されることがあるため、サービスを利用するクライアントは切断の発生を考慮してコードを作成するべきとされています。また、われわれのプログラム自身も接続を閉じることがあります。接続が終了した場合には、再接続する必要があります。
*/

type tweet struct {
	Text string
}

func readFromTwitter(votes chan<- string) {
	options, err := loadOptions()
	if err != nil {
		log.Println("選択肢の読み込みに失敗しました：", err)
		return
	}

	u, err := url.Parse("https://stream.twitter.com/1.1/statuses/filter.json")
	if err != nil {
		log.Println("URLの解析に失敗しました：", err)
		return
	}

	query := make(url.Values)
	query.Set("track", strings.Join(options, ","))

	req, err := http.NewRequest("POST", u.String(), strings.NewReader(query.Encode()))
	if err != nil {
		log.Println("検索のリクエストの作成に失敗しました：", err)
		return
	}

	resp, err := makeRequest(req, query)
	if err != nil {
		log.Println("検索のリクエストに失敗しました：", err)
		return
	}

	reader = resp.Body
	decoder := json.NewDecoder(reader)
	for {
		var tweet tweet
		if err := decoder.Decode(&tweet); err != nil {
			break
		}
		for _, option := range options {
			if strings.Contains(strings.ToLower(tweet.Text), strings.ToLower(option)) {
				log.Println("投票：", option)
				votes <- option
			}
		}
	}
}
