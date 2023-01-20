package znet

import (
	"fmt"
	"log"
	"myzinx/ziface"
	"net"
)

type Connection struct {
	Conn      *net.TCPConn      //TCP套接字
	ConnID    uint32            //连接ID
	isClosed  bool              //当前连接状态
	handleAPI ziface.HandleFunc //当前连接所绑定的处理业务的方法API
	Router    ziface.IRouter    //该连接的处理方法router
	ExitChan  chan bool         //告知当前连接已经退出/停止 channel

}

// 实例创建
func NewConnection(conn *net.TCPConn, connID uint32, callback_api ziface.HandleFunc) *Connection {
	return &Connection{
		Conn:      conn,
		ConnID:    connID,
		handleAPI: callback_api,
		isClosed:  false,
		ExitChan:  make(chan bool, 1),
	}
}
func (c *Connection) StartReader() {
	log.Println("Reader Groutine is runing...")
	defer log.Println("connID = ", c.ConnID, "Reader is exit,remot adder is ", c.RemoteAddr().String())
	defer c.Stop()
	for {
		buf := make([]byte, 512)
		cnt, err := c.Conn.Read(buf)
		if err != nil {
			log.Println("recv buf err", buf)
			c.ExitChan <- true
			continue
		}

		//得到当前客户端请求的Request数据
		req := Request{
			conn: c,
			data: buf,
		}
		//从路由Routers 中找到注册绑定Conn的对应Handle
		go func(request ziface.IRequest) {
			c.Router.PreHandle(request)
			c.Router.Handle(request)
			c.Router.PostHandle(request)
		}(&req)

		//调用当前连接所绑定的HandleAPI
		if err := c.handleAPI(c.Conn, buf, cnt); err != nil {
			fmt.Println("ConnID", c.ConnID, "handle is error", err)
			break
		}
	}
}

// 启动连接
func (c *Connection) Start() {
	log.Println("Conn Start()...Connid = ", c.ConnID)
	//启动从当前连接读数据的业务
	go c.StartReader()
	//TODO 启动从当前连接写数据的业务
}

// 停止连接，结束当前连接状态
func (c *Connection) Stop() {
	log.Println("Conn Stop()...ConnID = ", c.ConnID)
	//连接已关闭
	if c.isClosed == true {
		return
	}
	c.isClosed = true
	c.Conn.Close()
	close(c.ExitChan)
}

// Context() context.Context //返回ctx,用于用户自定义的协程获取连接退出状态
// 获取当前连接的绑定socket conn
func (c *Connection) GetTCPConnection() *net.TCPConn {
	return c.Conn
}

// 获取当前模块的连接ID
func (c *Connection) GetConnID() uint32 {
	return c.ConnID
}

// 获取远程客户端的TCP状态IP port
func (c *Connection) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// 发送数据，将数据发送给远程客户端
func (c *Connection) Send(data []byte) error {
	return nil
}
