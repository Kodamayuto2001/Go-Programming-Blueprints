package main

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
	creds *oauth.Credentials
)

func setupTwitterAuth() {
	var ts struct {
		ConsumerKey 	string 	`env:"SP_TWITTER_KEY,required"`
		ConsumerSecret	string 	`env:"SP_TWITTER_SECRET,required"`
		AccessToken		string	`env:"SP_TWITTER_ACCESSTOKEN,required"`
		AccessSecret	string	`env:"SP_TWITTER_ACCESSSECRET,required"`
	}

	if err := envdecode.Decode(&ts); err != nil {
		log.Fatalln(err)
	}

	creds = &oauth.Credentials{
		Token:	ts.AccessToken,
		Secret:	ts.AccessSecret,
	}

	authClient = &oauth.Client {
		Credentials: oauth.Credentials{
			Token:	ts.ConsumerKey,
			Secret:	ts.ConsumerSecret,
		}
	}
}

var (
	authSetupOnce	sync.Once 
	httpClient		*http.Client
)

func makeRequest(req *http.Request, params url.Values) (*http.Response, error) {
	authSetupOnce.Do(func() {
		setupTwitterAuth()
		httpClient = &http.Client{
			Transport: &http.Transport {
				Dial: dial,
			},
		}
	})

	formEnc := params.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", strconv.Itoa(len(formEnc)))
	req.Header.Set("Authorization",authClient.AuthorizationHeader(creds, "POST", req.URL, params))
	return httpClient.Do(req)
}