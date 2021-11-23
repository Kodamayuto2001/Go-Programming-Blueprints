package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func exists(domain string) (bool, error) {
	const whoisServer string = "com.whois-servers.net"
	conn, err := net.Dial("tcp", whoisServer+":43")
	if err != nil {
		return false, err
	}
	//	この関数がどのように終了したか（成功、失敗、異常終了）にかかわらず、deferとともに指定されたコードは最後に必ず実行される
	defer conn.Close()

	conn.Write([]byte(domain + "\r\n"))
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		if strings.Contains(strings.ToLower(scanner.Text()), "no match") {
			return false, nil
		}
	}
	return true, nil
}

var marks = map[bool]string{true: "〇", false: "×"}

/*
	ほとんどのWHOISサーバーでは、過負荷の状態を避けるために何らかの方法でリクエストの制限を行っています。
	制限を受けないようにするために、クライアント側で処理のペースを落とすというのは理にかなっています。
	一方、これはユニットテストの際にも有意義です。
	テストのたびに読者のコンピュータのIPアドレスからWHOISサーバーへのアクセスが集中して発生するというのは望ましくないはずです。
	最も適切なのは、実際のWHOISサーバーの代わりに模擬的なレスポンスを返してくれるオブジェクトを用意することです。
*/

func main() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		domain := s.Text()
		fmt.Print(domain, " ")
		exist, err := exists(domain)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(marks[!exist])
		time.Sleep(1 * time.Second)
	}
}
