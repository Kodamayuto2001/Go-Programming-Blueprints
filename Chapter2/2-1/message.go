package main

import (
	"time"
)

//	messageは一つのメッセージを表します。
//	Name:		ユーザー名
//	Message:	メッセージ
//	When:		メッセージが送信された時刻
type message struct {
	Name    string
	Message string
	When    time.Time
}
