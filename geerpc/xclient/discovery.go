package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type SelectMode int

// 负载均衡策略，只实现两种较为简单的策略
const (
	RandomSelect SelectMode = iota
	RoundRobinSelect
)

// 抽象接口，服务发现
type Discovery interface {
	Refresh() error                      //从注册中心更新服务列表
	Update(server []string) error        //手动更新服务列表
	Get(mode SelectMode) (string, error) //根据负载均衡策略，选择一个服务实例
	GetAll() ([]string, error)           //返回所有服务实例
}

type MultiServersDiscovery struct {
	r       *rand.Rand //产生随机数的实例
	mu      sync.RWMutex
	servers []string
	index   int //记录Round Robin 算法已经轮询到的位置，为了避免每次从 0 开始，初始化时随机设定一个值。
}

// 创建服务发现结构体
func NewMultiServerDiscovery(servers []string) *MultiServersDiscovery {
	d := &MultiServersDiscovery{
		servers: servers,
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	//随即设置初始值
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}

//接口实现

// 对其没有意义，空实现
func (d *MultiServersDiscovery) Refresh() error {
	return nil
}

func (d *MultiServersDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}

// 根据负载均衡策略，返回一个服务器实例
func (d *MultiServersDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return d.servers[d.r.Intn(n)], nil
	case RoundRobinSelect:
		s:=d.servers[d.index%n]
		d.index=(d.index+1)%n
		return s,nil
	default:
		return "",errors.New("rpc discovery: not supported select mode")
	}
}


//返回所有服务器实例
func (d*MultiServersDiscovery) GetAll()([]string,error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	servers:=make([]string,len(d.servers),len(d.servers))
	copy(servers,d.servers)
	return servers,nil
}