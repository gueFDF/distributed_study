package gin

import (
	"net/http"
)

type HandlerFunc func(C *Context)

type Engin struct {
	router *router
}

func New() *Engin {
	return &Engin{router: newRouter()}
}

// 添加路由
func (engin *Engin) addRoute(method string, pattern string, handler HandlerFunc) {
	engin.router.addRouter(method,pattern,handler)
}

// GET请求
func (engin *Engin) GET(palette string, handler HandlerFunc) {
	engin.addRoute("GET", palette, handler)
}

// POST请求
func (engin *Engin) POST(palette string, handler HandlerFunc) {
	engin.addRoute("POST", palette, handler)
}

func (engin *Engin) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engin)
}

func (engin *Engin) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	con := newContext(w, req)
	engin.router.handle(con)
}
