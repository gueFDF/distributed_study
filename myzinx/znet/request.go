package znet

import "myzinx/ziface"

type Request struct {
	conn ziface.IConnection //已经和客户端建立好的连接
	msg  ziface.IMessage    //请求消息封装
}

// 获取请求连接信息
func (r *Request) GetConnection() ziface.IConnection {
	return r.conn
}

// 获取请求消息数据
func (r *Request) GetData() []byte {
	return r.msg.GetData()
}

// 获取请求的消息ID
func (r *Request) GetMsgId() uint32 {
	return r.msg.GetMsgId()
}


