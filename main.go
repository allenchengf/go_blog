package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var router = mux.NewRouter()
var db *sql.DB

func initDB() {

	var err error
	config := mysql.Config{
		User:                 "root",
		Passwd:               "root",
		Addr:                 "127.0.0.1:3306",
		Net:                  "tcp",
		DBName:               "goblog",
		AllowNativePasswords: true,
	}

	// 準備數據庫連接池
	db, err = sql.Open("mysql", config.FormatDSN())
	checkError(err)

	// 設置最大連接數
	db.SetMaxOpenConns(100)
	// 設置最大空閒連接數
	db.SetMaxIdleConns(25)
	// 設置每個鏈結的過期時間
	db.SetConnMaxLifetime(5 * time.Minute)

	// 嘗試連接
	err = db.Ping()
	checkError(err)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<h1>Hello, 歡迎來到 goblog！</h1>")
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "請聯絡 "+
		"<a href=\"mailto:andycc77e@gmail.com\">andycc77e@gmail.com</a>")
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "<h1>頁面未找到 :(</h1><p>如有疑惑，請聯繫我們。</p>")
}

// Article  對應一條文章數據
type Article struct {
	Title, Body string
	ID          int64
}

// Link 方法用來生成文章鏈結
func (a Article) Link() string {
	showURL, err := router.Get("articles.show").URL("id", strconv.FormatInt(a.ID, 10))
	if err != nil {
		checkError(err)
		return ""
	}
	return showURL.String()
}

func articlesShowHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 獲取URL參數
	id := getRouteVariable("id", r)

	// 2. 讀取對應的文章數據
	article, err := getArticleByID(id)

	// 3. 如果出現錯誤
	if err != nil {
		if err == sql.ErrNoRows {
			// 3.1 數據未找到
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 文章未找到")
		} else {
			// 3.2 數據庫錯誤
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 Internal server error")
		}
	} else {
		// 4. 讀取成功，顯示文章
		tmpl, err := template.ParseFiles("resources/views/articles/show.gohtml")
		checkError(err)

		err = tmpl.Execute(w, article)
		checkError(err)
	}
}

func articlesIndexHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 執行查詢語句，返回一個結果
	rows, err := db.Query("SELECT * from articles")
	checkError(err)
	defer rows.Close()

	var articles []Article
	//2. 循環讀取結果
	for rows.Next() {
		var article Article
		// 2.1 掃描每一行的結果並賦值到一個article 对象中
		err := rows.Scan(&article.ID, &article.Title, &article.Body)
		checkError(err)
		// 2.2 將 article 追加到 articles 的這個數組中
		articles = append(articles, article)
	}

	// 2.3 檢測遍歷時是否發生錯誤
	err = rows.Err()
	checkError(err)

	// 3. 加載模板
	tmpl, err := template.ParseFiles("resources/views/articles/index.gohtml")
	checkError(err)

	// 4. 渲染模板，將文章所有的數據傳輸進去渲染模板
	err = tmpl.Execute(w, articles)
	checkError(err)
}

type ArticlesFormData struct {
	Title, Body string
	URL         *url.URL
	Errors      map[string]string
}

func articlesStoreHandler(w http.ResponseWriter, r *http.Request) {
	title := r.PostFormValue("title")
	body := r.PostFormValue("body")

	errors := validateArticleFormData(title, body)

	// 檢查是否有錯誤
	if len(errors) == 0 {
		lastInsertID, err := saveArticleToDB(title, body)
		if lastInsertID > 0 {
			fmt.Fprint(w, "插入成功，ID 為"+strconv.FormatInt(lastInsertID, 10))
		} else {
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 Internal server error")
		}
	} else {
		storeURL, _ := router.Get("articles.store").URL()

		data := ArticlesFormData{
			Title:  title,
			Body:   body,
			URL:    storeURL,
			Errors: errors,
		}
		tmpl, err := template.ParseFiles("resources/views/articles/create.gohtml")
		if err != nil {
			panic(err)
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			panic(err)
		}
	}
}

func saveArticleToDB(title string, body string) (int64, error) {

	// 變量初始化
	var (
		id   int64
		err  error
		rs   sql.Result
		stmt *sql.Stmt
	)

	// 1. 獲取一個 prepare 聲明語句
	stmt, err = db.Prepare("INSERT INTO articles (title, body) VALUES(?,?)")
	// 例行的錯誤檢測
	if err != nil {
		return 0, err
	}

	// 2. 在此函數運行結束後關閉此語句，防止佔用SQL連接
	defer stmt.Close()

	// 3. 執行請求，傳參數進入綁定的內容
	rs, err = stmt.Exec(title, body)
	if err != nil {
		return 0, err
	}

	// 4. 插入成功的話，會返回自增ID
	if id, err = rs.LastInsertId(); id > 0 {
		return id, nil
	}

	return 0, err
}

func forceHTMLMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 設置標頭
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// 2. 繼續處理請求
		h.ServeHTTP(w, r)
	})
}

func removeTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 除首页以外，移除所有請求路徑後面的/
		if r.URL.Path != "/" {
			r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		}

		// 2. 將請求傳遞下去
		next.ServeHTTP(w, r)

	})
}

func articlesCreateHandler(w http.ResponseWriter, r *http.Request) {
	storeURL, _ := router.Get("articles.store").URL()
	data := ArticlesFormData{
		Title:  "",
		Body:   "",
		URL:    storeURL,
		Errors: nil,
	}
	tmpl, err := template.ParseFiles("resources/views/articles/create.gohtml")
	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		panic(err)
	}
}

func createTables() {
	createArticlesSQL := `CREATE TABLE IF NOT EXISTS articles(
    id bigint(20) PRIMARY KEY AUTO_INCREMENT NOT NULL,
    title varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
    body longtext COLLATE utf8mb4_unicode_ci
); `

	_, err := db.Exec(createArticlesSQL)
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func articlesEditHandler(w http.ResponseWriter, r *http.Request) {

	// 1. 獲取URL參數
	id := getRouteVariable("id", r)

	// 2. 讀取對應的文章數據
	article, err := getArticleByID(id)

	// 3. 如果出現錯誤
	if err != nil {
		if err == sql.ErrNoRows {
			// 3.1 數據未找到
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 文章未找到")
		} else {
			// 3.2 數據庫錯誤
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 Internal server error")
		}
	} else {
		// 4. 讀取成功，顯示表單
		updateURL, _ := router.Get("articles.update").URL("id", id)
		data := ArticlesFormData{
			Title:  article.Title,
			Body:   article.Body,
			URL:    updateURL,
			Errors: nil,
		}
		tmpl, err := template.ParseFiles("resources/views/articles/edit.gohtml")
		checkError(err)

		err = tmpl.Execute(w, data)
		checkError(err)
	}
}

func articlesUpdateHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 獲取URL參數
	id := getRouteVariable("id", r)

	// 2. 讀取對應的文章數據
	_, err := getArticleByID(id)

	// 3. 如果出現錯誤
	if err != nil {
		if err == sql.ErrNoRows {
			// 3.1 數據未找到
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 文章未找到")
		} else {
			// 3.2 數據庫錯誤
			checkError(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "500 Internal server error")
		}
	} else {
		// 4. 未出現錯誤

		// 4.1 表單驗證
		title := r.PostFormValue("title")
		body := r.PostFormValue("body")

		errors := validateArticleFormData(title, body)

		if len(errors) == 0 {

			// 4.2 表單驗證通過，更新數據

			query := "UPDATE articles SET title = ?, body = ? WHERE id = ?"
			rs, err := db.Exec(query, title, body, id)

			if err != nil {
				checkError(err)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, "500 Internal server error")
			}

			// √ 更新成功，跳轉到文章詳情頁
			if n, _ := rs.RowsAffected(); n > 0 {
				showURL, _ := router.Get("articles.show").URL("id", id)
				http.Redirect(w, r, showURL.String(), http.StatusFound)
			} else {
				fmt.Fprint(w, "您没有做任何更改！")
			}
		} else {

			// 4.3 表单验证不通过，显示理由

			updateURL, _ := router.Get("articles.update").URL("id", id)
			data := ArticlesFormData{
				Title:  title,
				Body:   body,
				URL:    updateURL,
				Errors: errors,
			}
			tmpl, err := template.ParseFiles("resources/views/articles/edit.gohtml")
			checkError(err)

			err = tmpl.Execute(w, data)
			checkError(err)
		}
	}
}

func getRouteVariable(parameterName string, r *http.Request) string {
	vars := mux.Vars(r)
	return vars[parameterName]
}

func getArticleByID(id string) (Article, error) {
	article := Article{}
	query := "SELECT * FROM articles WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&article.ID, &article.Title, &article.Body)
	return article, err
}

func validateArticleFormData(title string, body string) map[string]string {
	errors := make(map[string]string)
	// 验证标题
	if title == "" {
		errors["title"] = "标题不能为空"
	} else if utf8.RuneCountInString(title) < 3 || utf8.RuneCountInString(title) > 40 {
		errors["title"] = "标题长度需介于 3-40"
	}

	// 验证内容
	if body == "" {
		errors["body"] = "内容不能为空"
	} else if utf8.RuneCountInString(body) < 10 {
		errors["body"] = "内容长度需大于或等于 10 个字节"
	}

	return errors
}

func main() {
	initDB()
	createTables()
	router.HandleFunc("/", homeHandler).Methods("GET").Name("home")
	router.HandleFunc("/about", aboutHandler).Methods("GET").Name("about")

	router.HandleFunc("/articles/{id:[0-9]+}", articlesShowHandler).Methods("GET").Name("articles.show")
	router.HandleFunc("/articles", articlesIndexHandler).Methods("GET").Name("articles.index")
	router.HandleFunc("/articles", articlesStoreHandler).Methods("POST").Name("articles.store")
	router.HandleFunc("/articles/create", articlesCreateHandler).Methods("GET").Name("articles.create")
	router.HandleFunc("/articles/{id:[0-9]+}/edit", articlesEditHandler).Methods("GET").Name("articles.edit")
	router.HandleFunc("/articles/{id:[0-9]+}", articlesUpdateHandler).Methods("POST").Name("articles.update")

	// 自定義404頁面
	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	router.Use(forceHTMLMiddleware)

	http.ListenAndServe(":3000", removeTrailingSlash(router))
}
