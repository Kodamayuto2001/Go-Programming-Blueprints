package main

import (
	"errors"
)

//	ErrNoAvatarはAvatarのインスタンスがアバターのURLを返すことができない場合に発生するエラーです。
//	errors.Newで初期化されるので、このオブジェクトが生成されるのは1回だけ
//	エラーのオブジェクトへのポインタが渡されているだけであり負荷はとても低くなっている。
//	Javaの例外処理の仕組みでは、例外のオブジェクトが高いコストを伴って毎回生成され、処理のフローの一部として機能している。
var ErrNoAvatarURL = errors.New("chat: アバターのURLを取得できません。")

//	Avatarはユーザーのプロフィール画像を表す型です。
type Avatar interface {
	//	GetAvatarURLは指定されたクライアントのアバターのURLを返します。
	//	問題が発生した場合にはエラーを返します。
	//	とくにURLを取得できなかった場合にはErrNoAvatarURLを返します。
	//	[命名]Getというセット時は必要なければつけないほうが望ましい
	GetAvatarURL(c *client) (string, error)
}

type AuthAvatar struct{}

var UseAuthAvatar AuthAvatar

func (_ AuthAvatar) GetAvatarURL(c *client) (string, error) {
	if url, ok := c.userData["avatar_url"]; ok {
		if urlStr, ok := url.(string); ok {
			return urlStr, nil
		}
	}
	return "", ErrNoAvatarURL
}