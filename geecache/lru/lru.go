package lru

import "container/list"

type Cache struct {
	maxBytes   int64      //最大内存
	nbytes     int64      //当前已使用的内存
	ll         *list.List //go语言自带的双向循环链表
	catche     map[string]*list.Element
	OneEvicted func(key string, value Value) //条目被移除时的回调函数(可选择)
}

// 双向循环链表的节点数据类型
type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

// new a instance
func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:   maxBytes,
		ll:         list.New(),
		catche:     make(map[string]*list.Element),
		OneEvicted: onEvicted,
	}
}

// search
// 1.找到对应的节点
// 2.将该节点移动到队尾
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.catche[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// delete(缓存淘汰)
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.catche, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OneEvicted != nil {
			c.OneEvicted(kv.key, kv.value)
		}
	}

}

// add or modify
func (c *Cache) Add(key string, value Value) {
	//如果该缓存已存在，就进行修改
	if ele, ok := c.catche[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else { //如果不存在就添加该缓存
		ele := c.ll.PushFront(&entry{key, value})
		c.catche[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	//如果缓存到达上线，就进行缓存淘汰
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
