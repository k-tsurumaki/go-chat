package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/k-tsurumaki/go-chat/trace"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/objx"
	"github.com/stretchr/signature"
)

type templateHandler struct {
	once     sync.Once // 一度しか実行されないことを保証
	filename string
	templ    *template.Template
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ServeHTTP は HTTP リクエストを受け取ったときにテンプレートを描画して返す。
	// - テンプレートのパースは高コストなので sync.Once を使って一度だけ行う
	// - レンダリング時にテンプレートに渡すデータ（Host / UserData）を準備する
	// - 認証クッキーがあればデコードしてテンプレートに渡す

	// テンプレートを一度だけ読み込む（初回のリクエスト時にのみ実行）
	t.once.Do(func() {
		// templates/<filename> をパース。失敗したら panic する（Must）。
		t.templ =
			template.Must(template.ParseFiles(filepath.Join("templates",
				t.filename)))
	})

	// テンプレートに渡すデータを構築
	data := map[string]interface{}{
		"Host": r.Host, // クライアント側で WebSocket 接続先を組み立てるのに使用
	}

	// 認証クッキーが存在すればデコードして UserData として渡す
	if authCookie, err := r.Cookie("auth"); err == nil {
		// authCookie の値は base64 エンコードされた JSON になっている（objx を使用してパース）
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}

	// 構築したデータでテンプレートを実行してレスポンスに書き込む
	t.templ.Execute(w, data)
}
func main() {
	var addr = flag.String("addr", ":8080", "アプリケーションのアドレス")
	flag.Parse() // フラグを解釈

	// Gomniauthのセットアップ
	gomniauth.SetSecurityKey(signature.RandomKey(64))
	gomniauth.WithProviders(
		// facebook.New(
		// 	getEnvOrFatal("FACEBOOK_CLIENT_ID"),
		// 	getEnvOrFatal("FACEBOOK_CLIENT_SECRET"),
		// 	getEnvOrFatal("FACEBOOK_REDIRECT_URL"),
		// ),
		// github.New(
		// 	getEnvOrFatal("GITHUB_CLIENT_ID"),
		// 	getEnvOrFatal("GITHUB_CLIENT_SECRET"),
		// 	getEnvOrFatal("GITHUB_REDIRECT_URL"),
		// ),
		google.New(
			getEnvOrFatal("GOOGLE_CLIENT_ID"),
			getEnvOrFatal("GOOGLE_CLIENT_SECRET"),
			getEnvOrFatal("GOOGLE_REDIRECT_URL"),
		),
	)
	// r := newRoom(UseAuthAvatar)
	r := newRoom(UseGravatar)
	r.tracer = trace.New(os.Stdout)
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "auth",
			Value: "",
			Path: "/",
			MaxAge: -1,
		})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/room", r)

	// チャットルーム開始
	go r.run()

	// Webサーバー開始
	log.Println("Webサーバーを開始します。ポート: ", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
