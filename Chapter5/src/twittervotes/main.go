package main

import "log"

/*
	Twitter検索に使用する文字列つまり投票での選択しを取得するために、MongoDBに接続して問い合わせを行います。
	これらの関数はそれぞれ、ローカルで実行されるMongoDBインスタンスへの接続とその解除を行います。ここではmgoパッケージが使われており、データベース接続を表すmgo.Sessionオブジェクトがグローバル変数dbにセットされます。
*/

var db *mgo.Session

func dialdb() error {
	var err error
	log.Println("MongoDBにダイヤル中： localhost")
	db, err = mgo.Dial("localhost")
	return err
}

func closedb() {
	db.Close()
	log.Println("データベース接続が閉じられました")
}

/*
	投票を表すドキュメントに含まれているのは、Optionsつまり選択肢だけではありません。
	しかし、我々のプログラムでは選択肢しかりようしないため、poll構造体にこれ以上フィールドを加える必要はありません。
	db変数を使い、ballotsデータベースに含まれるコレクションpollsを取り出します。
	そして、mgoパッケージの「流れるようなインターフェース」に基づいて、Findメソッドを使って検索を行います。
	ここでのnilはフィルタリングを行わないという意味です。

	※流れるようなインターフェース
	流れるようなインターフェース(fluent interface。名付け親はEric EvansとMartin Fowlerです)とは、メソッド呼び出しを連鎖させることによってコードを読みやすくできるようなAPI設計を意味します。ここでそれぞれのメソッドは、コンテキストとなるオブジェクト自身を返します。つまり、返されたオブジェクトに対して別のメソッドを直接呼び出すことができます。例えばmgoでは、下のようなコードを記述できます。
		query := col.Find(q).Sort("field").Limit(10).Skip(10)

*/

type poll struct {
	Options []string
}

func loadOptions() ([]string, error) {
	var options []string
	iter := db.DB("ballots").C("polls").Find(nil).Iter()
	var p poll
	for iter.Next(&p) {
		options = append(options, p.Options...)
	}
	iter.Close()
	return options, iter.Err()
}

func main() {

}
