package singleflight

import "sync"

// 正在进行或已经结束的请求
type call struct {
	wg  sync.WaitGroup
	Val interface{}
	err error
}

// 管理不同请求,主数据结构
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// 保证相同的请求,fn只会被调用一次
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call) //延迟初始化
	}
	//相同的请求已经存在，没必要再次添加
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait() //等待请求完成
		return c.Val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	//这个其实就是,发起请求
	c.Val, c.err = fn()
	c.wg.Done() //请求结束

	g.mu.Lock()
	delete(g.m, key) //请求结束，删除
	g.mu.Unlock()
	return c.Val, c.err
}
