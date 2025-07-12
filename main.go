package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
		<html>
		<head>
			<title>Welcome to My Web Server</title>
		</head>
		<body>
			<h1>Hello, World!</h1>
			<p>This is a simple web server written in Go.</p>
			<p>Visit <a href="https://www.example.com">Example</a> for more information.</p>
		</body>
		</html>
		`))
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
