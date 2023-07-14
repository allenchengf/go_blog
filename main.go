package main

import (
	"fmt"
	"net/http"
)

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.URL.Path == "/" {
		fmt.Fprint(w, "<h1>Hello, 這裡是 goblog</h1>")
	} else if r.URL.Path == "/about" {
		fmt.Fprint(w, "請聯絡我 "+
			"<a href=\"mailto:andycc77e@gmail.com\">andycc77e@gmail.com</a>")
	} else {
		fmt.Fprint(w, "<h1>請求頁面未找到 :(</h1>"+
			"<p>如有疑惑，請聯絡我們。</p>")
	}
}

func main() {
	http.HandleFunc("/", handlerFunc)
	http.ListenAndServe(":3000", nil)
}
