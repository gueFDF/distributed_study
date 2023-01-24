// 实现服务模块
package znet

import (
	"errors"
	"fmt"
	"log"
	"myzinx/utils"
	"myzinx/ziface"
	"net"
	"time"
)

type Server struct {
	//服务器名称
	Name string
	//tcp4 or other
	IPVersion string
	//服务绑定的IP地址
	IP string
	//服务绑定的端口
	Port int
	//当前Server的消息管理模块，用来绑定MsgId和对应的处理方法
	msgHandler ziface.IMsgHandle
}

func NewServer(name string) ziface.IServer {
	//先初始化全局配置文件
	utils.GlobalObject.Reload()
	return &Server{
		Name:       utils.GlobalObject.Name,
		IPVersion:  utils.GlobalObject.Version,
		IP:         utils.GlobalObject.Host,
		Port:       utils.GlobalObject.TcpPort,
		msgHandler: NewMsgHandle(),
	}
}

// 默认回调函数
func defaultcall_back(conn *net.TCPConn, data []byte, cnt int) error {
	log.Println("[Conn Handle] CallbackToClient...")
	if _, err := conn.Write(data[:cnt]); err != nil {
		log.Println("write back buf err", err)
		return errors.New("CallBackToClien error")
	}
	return nil
}

// 开启网络服务
func (s *Server) Start() {
	fmt.Printf("[START] Server listenner at IP:%s,Port:%d,is starting\n", s.IP, s.Port)
	//开启一个协程做服务器的Linster业务
	go func() {
		//获取TCP addr(转换格式)
		addr, err := net.ResolveTCPAddr(s.IPVersion, fmt.Sprintf("%s:%d", s.IP, s.Port))
		if err != nil {
			fmt.Println("resolve tcp addr err: ", err)
		}
		//监听
		listenner, err := net.ListenTCP(s.IPVersion, addr)
		if err != nil {
			fmt.Println("listen", s.IPVersion, "err", err)
			return
		}

		//已经开始监听
		fmt.Println("start Zinx server ", s.Name, " succ, now listening...")
		var cid uint32
		cid = 0
		//启动server网络连接业务
		for {
			conn, err := listenner.AcceptTCP()
			if err != nil {
				fmt.Println("Accept err ", err)
				continue
			}
			//TODO Server.Start() 设置服务器最大连接控制,如果超过最大连接，那么则关闭此新的连接
			//TODO Server.Start() 处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的
			//封装成一个连接模块
			dealConn := NewConnection(conn, cid, s.msgHandler)
			go dealConn.Start()
			cid++
		}
	}()
}

func (s *Server) Stop() {
	fmt.Println("[STOP] Zinx server , name ", s.Name)

	//TODO  Server.Stop() 将其他需要清理的连接信息或者其他信息 也要一并停止或者清理
}

func (s *Server) Serve() {
	s.Start()
	//TODO Server.Serve() 是否在启动服务的时候 还要处理其他的事情呢 可以在这里添加
	//阻塞，否则主协程会退出，子协程也会退出
	for {
		time.Sleep(10 * time.Second)
	}
}

func (s *Server) AddRouter(msdId uint32, router ziface.IRouter) {
	s.msgHandler.AddRouter(msdId, router)
}
