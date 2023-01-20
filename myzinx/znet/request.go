package znet

import "myzinx/ziface"

type Request struct {
	conn ziface.IConnection //已经和客户端建立好的连接
	data []byte             //客户端请求连接
}

// 获取请求连接信息
func (r *Request) GetConnection() ziface.IConnection {
	return r.conn
}

// 获取请求消息数据
func (r *Request) GetData() []byte {
	return r.data
}
