package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// 注册中心类
// 1.添加server
// 2.心跳机制保活
type GeeRegistry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr  string    //eg tcp@127.0.0.1:9999
	start time.Time //服务开启时间
}

const (
	defaultPath    = "/_geerpc_/registry"
	defaultTimeout = time.Minute * 5
)

// 创建注册中心实例，设置超时
func New(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

// 默认
var DefaultGeeRegister = New(defaultTimeout)

// 添加服务实例，如果服务已存在，更新开始时间
func (r *GeeRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{
			Addr:  addr,
			start: time.Now(),
		}
	} else {
		s.start = time.Now()
	}
}

// 返回可用的服务，若超时，删除该服务
func (r *GeeRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	for addr, s := range r.servers {
		//判断是超时，若没有添加到alive里面
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else { //超时了，从servers中删除
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

func (r *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		w.Header().Set("X-Geerpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		addr := req.Header.Get("X-Geerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path", registryPath)
}

func HandleHTTP() {
	DefaultGeeRegister.HandleHTTP(defaultPath)
}


// 服务启动，定时发送心跳，默认周期比注册中心设置的过期时间少一分钟
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		//每过一个周期发送一次心跳包
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

//发送心跳包
func sendHeartbeat(registry, addr string) error {
	log.Println(addr,"send heart beat to registyry",registry)
	httpClient:=&http.Client{}
	req,_:=http.NewRequest("POST",registry,nil)
	req.Header.Set("X-Geerpc-Server",addr)
	if _,err :=httpClient.Do(req);err!=nil {
		log.Println("rpc server: heart beat err:",err)
		return err
	}
	return nil
}

