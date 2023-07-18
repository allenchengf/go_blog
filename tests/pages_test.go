package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHomePage(t *testing.T) {
	baseURL := "http://localhost:3000"

	// 1. 請求--模擬用戶訪問瀏覽器請求
	var (
		resp *http.Response
		err  error
	)
	resp, err = http.Get(baseURL + "/")

	// 2. 檢測 —— 是否無錯且回應 200
	assert.NoError(t, err, "有錯誤發生，err 不為空")
	assert.Equal(t, 200, resp.StatusCode, "應返回狀態碼 200")
}
