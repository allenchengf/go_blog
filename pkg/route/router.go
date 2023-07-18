package route

import (
	"net/http"

	"github.com/gorilla/mux"
)

var Router = mux.NewRouter()

// Initialize 初始化路由
func Initialize() {
	Router = mux.NewRouter()
}

// RouteName2URL 通過路由名稱來獲取URL
func RouteName2URL(routeName string, pairs ...string) string {
	url, err := Router.Get(routeName).URL(pairs...)
	if err != nil {
		// checkError(err)
		return ""
	}

	return url.String()
}

// GetRouteVariable 獲取 URI 路由參數
func GetRouteVariable(parameterName string, r *http.Request) string {
	vars := mux.Vars(r)
	return vars[parameterName]
}
