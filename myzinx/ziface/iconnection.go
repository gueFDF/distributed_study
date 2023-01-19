package ziface

import "net"

// 定义连接接口抽象
type IConnection interface {
	Start() //启动连接
	Stop()  //停止连接，结束当前连接状态
	//Context() context.Context //返回ctx,用于用户自定义的协程获取连接退出状态
	//获取当前连接的绑定socket conn
	GetTCPConnection() *net.TCPConn
	GetConnID() uint32      //获取当前模块的连接ID
	RemoteAddr() net.Addr   //获取远程客户端的TCP状态IP port
	Send(data []byte) error //发送数据，将数据发送给远程客户端
}

// 定义一个处理连接业务的方法
type HandleFunc func(*net.TCPConn, []byte, int) error
