package main

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMessage = 64 << 10
)

// clientはチャットを行っている一人
type client struct {
	//	socketはこのクライアントのためのWebSocketです。
	socket *websocket.Conn
	//	sendはメッセージが送られるチャネルです。
	send chan *message
	//	roomはこのクライアントが参加しているチャットルームです。
	room *room
	//	userDataはユーザーに関する情報を保持します
	userData map[string]interface{}
}

func (c *client) read() {
	c.socket.SetReadLimit(maxMessage)
	if err := c.socket.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	c.socket.SetPongHandler(func(string) error {
		return c.socket.SetReadDeadline(time.Now().Add(pongWait))
	})
	defer c.socket.Close()

	for {
		var msg message
		//	WebSocketのReadJSONメソッドを利用して、message型をデコードする。
		if err := c.socket.ReadJSON(&msg); err == nil {
			msg.When = time.Now()
			msg.Name = c.userData["name"].(string)
			// msg.AvatarURL, _ = c.room.avatar.GetAvatarURL(c)

			if avatarURL, ok := c.userData["avatar_url"]; ok {
				msg.AvatarURL = avatarURL.(string)
			}
			c.room.forward <- &msg
		} else {
			return
		}
	}
}

func (c *client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.socket.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if err := c.socket.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				_ = c.socket.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(writeWait))
				return
			}
			//	WebSocketのWriteJSONメソッドを利用して、エンコードを行う。
			if err := c.socket.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.socket.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}
}
