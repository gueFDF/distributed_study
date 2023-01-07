package geerpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const MagicNumber = 0x3bef5c //3927900

type Option struct {
	MagicNumber    int
	CodcType       codec.Type //客户端可以选择
	ConnectTimeout time.Duration
	HandleTimeout  time.Duration
}

// 默认选项
var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodcType:       codec.GobType,
	ConnectTimeout: time.Second * 10, //默认超时时长为10s
}

// rpc server
type Server struct {
	serviceMap sync.Map
}

// return a new Server
func NewServer() *Server {
	return &Server{}
}

// 默认server实例
var DefaultServer = NewServer()

// 在监听器上接受连接并为请求提供服务
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error", err)
			return
		}
		go server.ServerCon(conn)
	}

}

func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

/*
| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
*/

//最前面是JSON编码Option,后面Header和Body的编码格式由option的中的CodecType决定

func (srever *Server) ServerCon(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option

	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: option error:", err)
		return
	}

	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server:invalid magic number %x", opt.MagicNumber)
		return
	}

	//根据编码累类型获取相应函数
	f := codec.NewCodecFuncMap[opt.CodcType]
	if f == nil {
		log.Printf("rpc server:invaild codec type %s", opt.CodcType)
		return
	}

	srever.serverCodec(f(conn), &opt)

}

// 无效请求，在发生错误响应时起占位作用
var invalidRequest = struct{}{}

/*sreverCodec
  1.读取请求readRequest
  2.处理请求handleRequest
  3.回复请求sendRequest
*/

func (server *Server) serverCodec(cc codec.Codec, opt *Option) {
	sending := new(sync.Mutex) //回复响应时并行回复
	wg := new(sync.WaitGroup)
	for {
		//读取请求
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break //无法恢复，关闭连接
			}
			req.h.Error = err.Error()
			//回复请求
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		//处理请求
		go server.handleRequest(cc, req, sending, wg, opt.HandleTimeout)
	}

	wg.Wait() //等待所有request处理完成才close
	_ = cc.Close()
}

// 存放一次请求的所有信息
type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
	mtype        *methodType
	svc          *service
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc serve:read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}

	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	//保证argvi是一个pointer,readbody需要一个pointer作为参数
	//参数可能是一个pointer也可能不是pointer
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	// //通过反射机制创建实际类型
	// req.argv = reflect.New(reflect.TypeOf(""))

	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read argv err:", err)
		return req, nil
	}
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	//因为客户端只是一个，为了防止客户端数据接收混乱，所以需要加锁
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server:write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	//应根据请求类型执行相应的Rpc方法（目前暂未实现服务注册，打印一句话来代替）

	defer wg.Done()
	called := make(chan struct{})
	sent := make(chan struct{})
	//交给子协程去去调用方法，这样就可以处理调用超时了，当超市，主协程直接退出
	go func() {
		err := req.svc.call(req.mtype, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
			sent <- struct{}{}
			return
		}
		server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
		sent <- struct{}{}
	}()

	//主协程进行监控
	//没有设置超时，就阻塞等待完成后返回
	if timeout == 0 {
		<-called
		<-sent
	}

	select {
	//超时
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		server.sendResponse(cc, req.h, invalidRequest, sending)
	case <-called:
		//调用完成，等待完成sendrequest
		<-sent
	}
}

// 注册服务
func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)

	//注册到map当中
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc:service already defined" + s.name)
	}
	return nil
}

func Register(rcvr interface{}) error { return DefaultServer.Register(rcvr) }

// findService 通过serviceMethod从serviceMap中找到对应的service
func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}
	//获取服务名和方法名
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server:can't find service " + serviceName)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server:can't find method" + methodName)
	}
	return
}
