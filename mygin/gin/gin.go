package gin

import (
	"net/http"
)

type HandlerFunc func(c *Context)

// 继承RouterGroups所有能力
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup
}

type RouterGroup struct {
	prefix     string        //公共前缀
	middleware []HandlerFunc //支持中间件
	parent     *RouterGroup  //支持嵌套,父亲组
	engine     *Engine       //所有组共享一个engine实例
}

func New() *Engine {
	Engine := &Engine{router: newRouter()}
	Engine.RouterGroup = &RouterGroup{engine: Engine}
	Engine.groups = []*RouterGroup{Engine.RouterGroup}
	return Engine
}

// 创建一个新组
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}

	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// 添加路由
func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	group.engine.router.addRoute(method, pattern, handler)
}

// GET请求
func (group *RouterGroup) GET(palette string, handler HandlerFunc) {
	group.addRoute("GET", palette, handler)
}

// POST请求
func (group *RouterGroup) POST(palette string, handler HandlerFunc) {
	group.addRoute("POST", palette, handler)
}

func (Engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, Engine)
}

func (Engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	con := newContext(w, req)
	Engine.router.handle(con)
}
