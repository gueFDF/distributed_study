package znet

import (
	"errors"
	"io"
	"log"
	"myzinx/ziface"
	"net"
)

type Connection struct {
	Conn       *net.TCPConn      //TCP套接字
	ConnID     uint32            //连接ID
	isClosed   bool              //当前连接状态
	MsgHandler ziface.IMsgHandle //消息处理模块
	ExitChan   chan bool         //告知当前连接已经退出/停止 channel
	msgChan    chan []byte       //用于读和写分离
}

// 实例创建
func NewConnection(conn *net.TCPConn, connID uint32, msgHandler ziface.IMsgHandle) *Connection {
	return &Connection{
		Conn:       conn,
		ConnID:     connID,
		isClosed:   false,
		ExitChan:   make(chan bool, 1),
		MsgHandler: msgHandler,
		msgChan:    make(chan []byte),
	}
}

// 写协程
func (c *Connection) startWriter() {
	log.Println("[Writer Goroutine is running]")
	defer log.Println(c.RemoteAddr().String(), "[conn Writer exit]")

	for {
		select {
		case date := <-c.msgChan:
			//有数据要写给客户端
			if _, err := c.Conn.Write(date); err != nil {
				log.Println("Send Data error:, ", err, " Conn Writer exit")
				return
			}
		case <-c.ExitChan:
			//关闭
			return
		}
	}
}

// 读协程
func (c *Connection) StartReader() {
	log.Println("Reader Groutine is runing...")
	defer log.Println(c.RemoteAddr().String(), "[conn Reader exit]")
	defer c.Stop()
	for {
		dp := NewDataPack()
		//拆包
		msg, err := dp.Unpack(c.GetTCPConnection())
		if err != nil {
			log.Println("unpack error ", err)
			c.ExitChan <- true
			if err==io.EOF {
				break
			}
			continue
		}
		//得到当前客户端请求的Request数据
		req := Request{
			conn: c,
			msg:  msg, //将之前的buf 改成 msg
		}
		//从路由Routers 中找到注册绑定Conn的对应Handle
		go c.MsgHandler.DoMsgHandle(&req)
	}
}

// 启动连接
func (c *Connection) Start() {
	log.Println("Conn Start()...Connid = ", c.ConnID)
	//启动从当前连接读数据的业务
	go c.StartReader()
	//启动从当前连接写数据的业务
	go c.startWriter()
	for {
		select {
		case <-c.ExitChan:
			//退出消息，不再阻塞
			return
		}
	}
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

// 直接将Message数据发送给远程的TCP客户端
func (c *Connection) SendMsg(msgID uint32, data []byte) error {
	if c.isClosed == true {
		return errors.New("Connection closed when send msg")
	}
	//将data封包
	dp := NewDataPack()
	msg, err := dp.Pack(NewMsgPackage(msgID, data))
	if err != nil {
		log.Println("Pack error msg id = ", msgID)
		return errors.New("Pack error msg")
	}
	//写回客户端
	c.msgChan <- msg
	return nil
}
