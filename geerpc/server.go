package geerpc

import (
	"encoding/json"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c //3927900

type Option struct {
	MagicNumber int
	CodcType    codec.Type //客户端可以选择
}

// 默认选项
var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodcType:    codec.GobType,
}

// rpc server
type Server struct{}

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

	srever.serverCodec(f(conn))

}

// 无效请求，在发生错误响应时起占位作用
var invalidRequest = struct{}{}

/*sreverCodec
  1.读取请求readRequest
  2.处理请求handleRequest
  3.回复请求sendRequest
*/

func (server *Server) serverCodec(cc codec.Codec) {
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
		go server.handleRequest(cc, req, sending, wg)
	}

	wg.Wait() //等待所有request处理完成才close
	_ = cc.Close()
}


//存放一次请求的所有信息
type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

func (server*Server) readRequestHeader(cc codec.Codec)(*codec.Header,error) {
	var h codec.Header
	if err:=cc.ReadHeader(&h);err!=nil{
		if err!=io.EOF &&err!=io.ErrUnexpectedEOF{
			log.Println("rpc serve:read header error:",err)
		}
		return nil ,err
	}
	return &h,nil
}


func (servre*Server)readRequest(cc codec.Codec) (*request,error) {
	h,err:=servre.readRequestHeader(cc)
	if err!=nil{
		return nil,err
	}
	req:=&request{h:h}

	//通过反射机制创建实际类型
	req.argv=reflect.New(reflect.TypeOf(""))

	if err=cc.ReadBody(req.argv.Interface());err!=nil {
		log.Println("rpc server: read argv err:",err)
	}
	return req,nil
}


func (server*Server)sendResponse(cc codec.Codec,h*codec.Header,body interface{},sending*sync.Mutex) {
	//因为客户端只是一个，为了防止客户端数据接收混乱，所以需要加锁
	sending.Lock()
	defer sending.Unlock()
	if err:=cc.Write(h,body);err!=nil {
		log.Println("rpc server:write response error:",err)
	}
}


func(server*Server)handleRequest(cc codec.Codec,req*request,sending*sync.Mutex,wg*sync.WaitGroup){
	//应根据请求类型执行相应的Rpc方法（目前暂未实现服务注册，打印一句话来代替）

	defer wg.Done()
	log.Println(req.h,req.argv.Elem())

	req.replyv=reflect.ValueOf(fmt.Sprintf("geerpc resp %d",&req.h.Seq))
	server.sendResponse(cc,req.h,req.replyv.Interface(),sending)
}