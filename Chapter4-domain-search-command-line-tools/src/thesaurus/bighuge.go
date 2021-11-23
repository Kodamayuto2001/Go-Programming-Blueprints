package thesaurus

import (
	"encoding/json"
	"fmt"
	"net/http"
)

/*
	JSONでは、全て小文字の名前が使われるのが一般的ですが、Goのプログラムでは戦闘が大文字の名前を使う必要があります。
	それぞれのフィールドをエクスポートし、encoding/jsonパッケージに存在を知らせる必要があるためです。
	フィールド名がすべて小文字だと、encoding/jsonパッケージに無視されてしまいます。
	ただし、synonymsとwordsという2つの型自体についてはエクスポートの必要はありません。
*/

type BigHuge struct {
	APIKey string
}

//	類義語
type synonyms struct {
	//	名詞
	Noun *words `json:"noun"`
	//	動詞
	Verb *words `json:"verb"`
}

type words struct {
	Syn []string `json:"syn"`
}

//	APIにアクセスして受け取ったレスポンスを解釈して返すSynonymsメソッド
func (b *BigHuge) Synonyms(term string) ([]string, error) {
	var syns []string
	response, err := http.Get("https://words.bighugelabs.com/api/2/" + b.APIKey + "/" + term + "/json")
	if err != nil {
		//	%v	全ての型に使えるverb	値をデフォルトのフォーマットでの表現を出力する。
		//	%q 	対応する文字をシングルクォートで囲んだ文字列　Goの文法上のエスケープをした文字列
		return syns, fmt.Errorf("bighuge: %qの類語検索に失敗しました： %v", term, err)
	}
	var data synonyms
	defer response.Body.Close()
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return syns, err
	}
	syns = append(syns, data.Noun.Syn...)
	syns = append(syns, data.Verb.Syn...)
	return syns, nil
}
