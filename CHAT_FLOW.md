# チャット動作フロー

ブラウザが `http://localhost:8080/chat` にアクセスしてからチャットが動作するまでの一連の流れを、関係するファイルと該当箇所を明記してまとめたドキュメントです。

---

## 概要

1. ブラウザが `http://localhost:8080/chat` を開く → 認証確認 → `chat.html` を返す
2. `chat.html` の JS がページ読み込み時に `ws://<Host>/room` へ WebSocket 接続を開始（初回は HTTP ハンドシェイク／Upgrade）
3. サーバ側の `/room` ハンドラ（`room.ServeHTTP`）で接続を Upgrade、`client` を生成して `room` に参加させる
4. `room.run` がメッセージ中継の中心となり、`forward` → 各 `client.send` に配信する
5. 各 `client.write()` が WebSocket 経由でブラウザにメッセージを送り、ブラウザ側で表示される

---

## 登場する主なファイルと関数（参照箇所）

- ルーティング／テンプレート:
  - `main.go` — `http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))`、`templateHandler.ServeHTTP`
- 認証ミドルウェア:
  - `auth.go` — `MustAuth`（`authHandler.ServeHTTP`）、`loginHandler`
- クライアントテンプレート（フロントエンド）:
  - `templates/chat.html` — ページ読み込み時の JS、`new WebSocket("ws://{{.Host}}/room")`、`socket.send()`／`socket.onmessage`
- ルーム（ハブ）:
  - `room.go` — `room.ServeHTTP`（Upgrade と `client` の生成）、`room.run`（join/leave/forward のループ）
- 接続ごとの処理:
  - `client.go` — `client.read()`（受信→`room.forward`）、`client.write()`（`c.send` からブラウザへ送信）

---

## 詳細な処理の流れ（ステップごと）

### 1) ブラウザが `GET /chat`

- `main.go` のルーティングで `MustAuth(...)` が適用される。
- `auth.go` の `authHandler.ServeHTTP` が認証クッキー `auth` を確認:
  - 無ければ `/login` にリダイレクト。
  - ある場合は `templateHandler.ServeHTTP` に処理を渡す。
- `templateHandler.ServeHTTP` (`main.go`) はテンプレートを1回だけ読み込み（`sync.Once`）、`data`（`Host`、`UserData`）をテンプレートに渡して `chat.html` を返す。

### 2) ブラウザ側の初期化（`templates/chat.html`）

- ページの JavaScript が起動し、`new WebSocket("ws://{{.Host}}/room")` を実行して WebSocket の接続を開始。
- この時点でブラウザは `/room` に対して WebSocket ハンドシェイク（Upgrade）を送る。

### 3) サーバ側での接続確立（`room.ServeHTTP`）

- `upgrader.Upgrade(w, req, nil)` によりリクエストを WebSocket にアップグレード、`*websocket.Conn` を取得。
- リクエストの `auth` クッキーを読み、`userData` を復元する。
- `client` 構造体を生成（`socket`, `send` チャネル, `room`, `userData`）。
- `r.join <- client` でルームに参加を通知。
- `defer r.leave <- client` で関数終了時に退出を通知するようにする。
- `go client.write()` を開始（ルームから来るメッセージをブラウザに送る goroutine）。
- `client.read()` を呼んでブロッキングでクライアントからのメッセージを読み続ける。

### 4) ルーム側の中継（`room.run`）

- `run()` は `join`, `leave`, `forward` の3チャネルを `select` で監視する常駐 goroutine（`go r.run()` は `main` で起動）。
- `join` を受けると `clients[client] = true`（参加追加）。
- `leave` を受けると `delete(r.clients, client)` と `close(client.send)`（クリーンアップ）。
- `forward` を受けるとメッセージを全 `clients` に流す。送れないクライアントは切断してクリーンアップする。

### 5) 実際のメッセージ送受信の流れ

- ブラウザの送信:
  - ユーザがフォーム送信 → JS が `socket.send(JSON.stringify({...}))` を呼ぶ。
- サーバの受信:
  - その接続の `client.read()` が `ReadJSON(&msg)`（または `ReadMessage` で解析）で受信 → `msg.When = time.Now()`、`msg.Name = c.userData["name"]` を付与 → `c.room.forward <- msg`
- ルーム中継:
  - `room.run` が `forward` 受信 → 各 `client.send <- msg` を試みる
- 各クライアント送信:
  - `client.write()` が `for msg := range c.send { c.socket.WriteJSON(msg) }` で送信 → ブラウザの `socket.onmessage` が受信して DOM に表示

### 6) 切断とクリーンアップ

- `ReadJSON` / `WriteJSON` がエラーになったら、`client.read()` / `client.write()` はループを抜け `c.socket.Close()` を呼ぶ。
- `room.ServeHTTP` の `defer r.leave <- client` により `room.run` に退出が通知され、`clients` から削除・チャネルを閉じる。
- `client.write()` 側は `close(client.send)` により `for range` が終わり goroutine が終了する。

---

## 注意点・実装チェックリスト

- JSON フィールドの整合性
  - フロントエンド（`chat.html`）が送る JSON 形式と、サーバの `message` 型が一致しているか確認（例: `{"Message": "..."}`  vs `message.Message`）。
- 認証情報の取り扱い
  - `auth` クッキーの内容（base64 で埋めた objx データ）は `room.ServeHTTP` とテンプレートの両方で使われる。整合を壊す変更に注意。
- エラーハンドリング
  - `ReadJSON` / `WriteJSON` のエラー時のログや原因の把握（タイムアウト、JSON parse error 等）を充実させるとデバッグが楽。
- WebSocket スキーム
  - `ws://` を使っているが、HTTPS 化する場合は `wss://` を使う必要がある（`chat.html` でプロトコル判定を入れることを推奨）。
- セキュリティ
  - 認証や CSRF、セッション管理、クッキーの Secure/HttpOnly 属性などは運用環境に合わせて設定する。

---

## 簡易シーケンス（要約）

- Browser -> GET /chat
  - Server: `MustAuth` → `templateHandler.ServeHTTP` → `chat.html`
- Browser (JS): new WebSocket("ws://Host/room")
- Browser -> /room (Upgrade)
  - Server: `room.ServeHTTP` → `upgrader.Upgrade` → create `client` → `r.join <- client` → `go client.write()` → `client.read()`
- User submits message -> Browser socket.send(JSON)
  - Server: `client.read()` -> `r.forward <- msg`
  - Server: `room.run()` broadcasts -> `client.send <- msg`
  - Server: each `client.write()` -> `WriteJSON` -> Browser `socket.onmessage` -> DOM 更新

---

## ローカルでの実行例

```bash
# 環境変数（例: .env で設定済みなら不要）
export GOOGLE_CLIENT_ID=...
export GOOGLE_CLIENT_SECRET=...
export GOOGLE_REDIRECT_URL=http://localhost:8080/auth/callback/google

# サーバ起動
go run .
# またはビルドして実行
go build -o go-chat
./go-chat
```

---

必要ならこのドキュメントを README に統合したり、図（シーケンス図）やログ出力を追加するパッチも作成できます。ご希望があれば教えてください。
