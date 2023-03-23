package gin

import (
	"net/http"
	"path"
	"strings"
	"text/template"
)

type HandlerFunc func(c *Context)

// 继承RouterGroups所有能力
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup
	//用于html渲染
	htmlTemplates *template.Template
	funcMap       template.FuncMap
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

func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middleware []HandlerFunc
	for _, group := range engine.groups {
		//同一个组的共享所有中间件
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middleware = append(middleware, group.middleware...)
		}
	}

	con := newContext(w, req)
	con.handlers = middleware
	con.engine = engine
	engine.router.handle(con)
}

// 添加中间件
func (group *RouterGroup) Use(middleware ...HandlerFunc) {
	group.middleware = append(group.middleware, middleware...)
}

// creat staic hanlder
func (group *RouterGroup) creatStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// serve static files
func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.creatStaticHandler(relativePath, http.Dir(root))

	urlPattern := path.Join(relativePath, "/*filepath")

	//注册 GET handlers
	group.GET(urlPattern, handler)
}

func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}
