package xclient

import (
	"context"
	. "geerpc"
	"reflect"
	"sync"
)

type XClient struct {
	d       Discovery
	mode    SelectMode
	opt     *Option
	mu      sync.Mutex
	clients map[string]*Client
}

func NewClient(d Discovery, mode SelectMode, opt *Option) *XClient {
	return &XClient{
		d:       d,
		mode:    mode,
		opt:     opt,
		clients: make(map[string]*Client),
	}
}

// 关闭所有Client
func (xc *XClient) Close() error {
	xc.mu.Lock()
	defer xc.mu.Unlock()

	for key, client := range xc.clients {
		_ = client.Close()
		delete(xc.clients, key)
	}
	return nil
}

func (xc *XClient) dial(rpcAddr string) (*Client, error) {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	client, ok := xc.clients[rpcAddr]

	//找到了，但是该客户端已关闭
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(xc.clients, rpcAddr)
	}
	//未发现该客户端，重新创建
	if client == nil {
		var err error
		//创建
		client, err = XDial(rpcAddr, xc.opt)
		if err != nil {
			return nil, err
		}
		//注册
		xc.clients[rpcAddr] = client
	}
	return client, nil
}

func (xc *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
	//寻找客户端
	client, err := xc.dial(rpcAddr)
	if err != nil {
		return err
	}
	return client.Call(ctx, serviceMethod, args, reply)
}

func (xc *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	//根据负载均衡策略，选择服务器
	rpcAddr, err := xc.d.Get(xc.mode)
	if err != nil {
		return err
	}

	return xc.call(rpcAddr, ctx, serviceMethod, args, reply)
}

// 广播功能
func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	//获取全部服务
	severs, err := xc.d.GetAll()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var e error

	replyDone := reply == nil //判断返回值是否需要设置
	ctx, cancel := context.WithCancel(ctx)

	for _, rpcAddr := range severs {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var clonedReply interface{}
			if reply!=nil {
				clonedReply=reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface()
			}
			err :=xc.call(rpcAddr,ctx,serviceMethod,args,clonedReply)
			mu.Lock()
			if err !=nil&&e==nil {
				e=err
				cancel()
			}
			if err==nil&&!replyDone{
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem())
				replyDone=true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}
