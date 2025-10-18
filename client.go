package main

import (
	"time"

	"github.com/gorilla/websocket"
)

type client struct {
	socket   *websocket.Conn
	send     chan *message
	room     *room
	userData map[string]interface{}
}

func (c *client) read() {
	// read はクライアント側から送られてくる JSON メッセージを受信して処理する
	// - 永久ループで WebSocket から JSON を読み込む
	// - 読み込めたら時刻と送信者名を設定してルームの forward チャネルへ流す
	// - エラー（接続切断等）が発生したらループを抜けてソケットを閉じる
	for {
		var msg *message
		if err := c.socket.ReadJSON(&msg); err == nil {
			// 受信メッセージに送信時刻とユーザ名を付与
			msg.When = time.Now()
			msg.Name = c.userData["name"].(string)
			// ルームに転送（ルーム側で全クライアントへ配信される）
			c.room.forward <- msg
		} else {
			// 読み取りに失敗したら（切断等）ループを抜ける
			break
		}
	}
	// 接続を閉じる
	c.socket.Close()
}

func (c *client) write() {
	// write はルームから送られてくるメッセージを受け取り WebSocket 経由でクライアントに送信する
	// - c.send チャネルをレンジして受信した message を JSON として書き出す
	// - 書き込みでエラーが起きたらループを抜けてソケットを閉じる
	for msg := range c.send {
		if err := c.socket.WriteJSON(msg); err != nil {
			break
		}
	}
	// 送信ループ終了時に接続を閉じる
	c.socket.Close()
}
