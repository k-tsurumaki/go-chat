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
	t.once.Do(func() {
		t.templ =
			template.Must(template.ParseFiles(filepath.Join("templates",
				t.filename)))
	})
	data := map[string]interface{}{
		"Host": r.Host,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
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
	r := newRoom()
	r.tracer = trace.New(os.Stdout)
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", r)

	// チャットルーム開始
	go r.run()

	// Webサーバー開始
	log.Println("Webサーバーを開始します。ポート: ", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
