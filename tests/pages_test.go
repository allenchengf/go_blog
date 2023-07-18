package tests

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllPages(t *testing.T) {

	baseURL := "http://localhost:3000"

	// 1. 聲明加初始化測試數據
	var tests = []struct {
		method   string
		url      string
		expected int
	}{
		{"GET", "/", 200},
		{"GET", "/about", 200},
		{"GET", "/notfound", 404},
		{"GET", "/articles", 200},
		{"GET", "/articles/create", 200},
		{"GET", "/articles/3", 200},
		{"GET", "/articles/3/edit", 200},
		{"POST", "/articles/3", 200},
		{"POST", "/articles", 200},
		{"POST", "/articles/1/delete", 404},
	}

	// 2. 請求以獲取響應
	for _, test := range tests {
		t.Logf("当前请求 URL: %v \n", test.url)
		var (
			resp *http.Response
			err  error
		)
		// 2.1 請求以獲取響應
		switch {
		case test.method == "POST":
			data := make(map[string][]string)
			resp, err = http.PostForm(baseURL+test.url, data)
		default:
			resp, err = http.Get(baseURL + test.url)
		}
		// 2.2 斷言
		assert.NoError(t, err, "請求 "+test.url+" 時報錯")
		assert.Equal(t, test.expected, resp.StatusCode, test.url+" 應返回狀態碼 "+strconv.Itoa(test.expected))
	}
}
