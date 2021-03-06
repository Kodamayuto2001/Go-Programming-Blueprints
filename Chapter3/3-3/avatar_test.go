package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	gomniauthtest "github.com/stretchr/gomniauth/test"
)

func TestAuthAvatar(t *testing.T) {
	//	Goではゼロ値による初期化が行われるので不逞な状態にはならない
	var authAvatar AuthAvatar

	testUser := &gomniauthtest.TestUser{}
	testUser.On("AvatarURL").Return("", ErrNoAvatarURL)
	testChatUser := &chatUser{User: testUser}
	url, err := authAvatar.GetAvatarURL(testChatUser)

	// client := new(client)
	// url, err := authAvatar.GetAvatarURL(client)

	if err != ErrNoAvatarURL {
		t.Error("値が存在しない場合、AuthAvatar.GetAvatarURLは、ErrNoAvatarURLを返すべきです")
	}

	//	値をセットします。
	testUrl := "http://url-to-avatar/"

	testUser = &gomniauthtest.TestUser{}
	testChatUser.User = testUser
	testUser.On("AvatarURL").Return(testUrl, nil)
	url, err = authAvatar.GetAvatarURL(testChatUser)

	// client.userData = map[string]interface{}{"avatar_url": testUrl}
	// url, err = authAvatar.GetAvatarURL(client)

	if err != nil {
		t.Error("値が存在する場合、AuthAvatar.GetAvatarURLはエラーを返すべきではありません")
	} else {
		if url != testUrl {
			t.Error("AuthAvatar.GetAvatarURLは正しいURLを返すべきです")
		}
	}
}

func TestGravatarAvatar(t *testing.T) {
	var gravatarAvatar GravatarAvatar
	// client := new(client)
	// // client.userData = map[string]interface{}{
	// // 	"email:" "MyEmailAddress@example.com"
	// // }
	// client.userData = map[string]interface{}{
	// 	"userid": "0bc83cb571cd1c50ba6f3e8a78ef1346",
	// }

	// url, err := gravatarAvatar.GetAvatarURL(client)

	user := &chatUser{uniqueID: "abc"}
	url, err := gravatarAvatar.GetAvatarURL(user)

	if err != nil {
		t.Error("GravatarAvatar.GetAvatarURLはエラーを返すべきではありません")
	}
	// if url != "//www.gravatar.com/avatar/0bc83cb571cd1c50ba6f3e8a78ef1346" {
	// 	t.Errorf("GravatarAvatar.GetAvatarURLが%sという誤った値を返しました", url)
	// }
	if url != "//www/gravatar.com/avatar/abc" {
		t.Errorf("GravatarAvatar.GetAvatarURLが%sという誤った値を返しました", url)
	}
}

func TestFileSystemAvatar(t *testing.T) {
	//	テスト用のアバターのファイルを生成します
	filename := filepath.Join("avatars", "abc.jpg")
	ioutil.WriteFile(filename, []byte{}, 0777)

	//	【参照】https://stackoverflow.com/questions/16008604/why-add-after-closure-body-in-golang
	defer func() { os.Remove(filename) }()

	var fileSystemAvatar FileSystemAvatar
	// client := new(client)
	// client.userData = map[string]interface{}{"userid": "abc"}
	// url, err := fileSystemAvatar.GetAvatarURL(client)

	user := &chatUser{uniqueID: "abc"}
	url, err := fileSystemAvatar.GetAvatarURL(user)

	if err != nil {
		t.Error("FileSystemAvatar.GetAvatarURLはエラーを返すべきではありません")
	}
	if url != "/avatars/abc.jpg" {
		t.Errorf("FileSystemAvatar.GetAvatarURLが%sという誤った値を返しました", url)
	}
}
