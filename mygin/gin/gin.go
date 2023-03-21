package gin

import (
	"fmt"
	"net/http"
)

type HandlerFunc func(http.ResponseWriter, *http.Request)

type Engin struct {
	router map[string]HandlerFunc
}

func New() *Engin {
	return &Engin{router: make(map[string]HandlerFunc)}
}

// 添加路由
func (engin *Engin) addRoute(method string, pattern string, handler HandlerFunc) {
	key := method + "-" + pattern
	engin.router[key] = handler
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
	key := req.Method + "-" + req.URL.Path
	if handler, ok := engin.router[key]; ok {
		handler(w, req)
	} else {
		fmt.Fprintf(w, "404 NOT FoUND: %s\n", req.URL)
	}
}
