package handler

import (
	"net/http"
	"fmt"
)
// HTTPInterceptor : http请求拦截器
func HTTPInterceptor(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(//hr这里（）可以看作就是把func变成type的一种类型转换
		func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			username := r.Form.Get("username")
			token := r.Form.Get("token")

			//验证登录token是否有效
			if len(username) < 3 || !IsTokenValid(username,token) {
				//w.Write([]byte("token校验失败或username<3"))
				// w.WriteHeader(http.StatusForbidden)
				// token校验失败则跳转到登录页面
				fmt.Println("token校验失败或username<3")
				http.Redirect(w, r, "/static/view/signin.html", http.StatusFound)
				return
			}
			h(w, r)
		})
}
