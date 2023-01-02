package geerpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"log"
	"net"
	"sync"
	"time"
)

// Call represnets an active RPC
type Call struct {
	Seq           uint64      //ID
	ServiceMethod string      //format "<service>.<method>"
	Args          interface{} //arguments
	Reply         interface{} //return value
	Error         error       //if err occurs,it will be set
	Done          chan *Call  //Strobes when call is complete
}

// 为了支持异步调用，当调用结束时，会调用call.done()
func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	cc       codec.Codec
	opt      *Option
	sending  sync.Mutex //用来保证请求有序发送
	header   codec.Header
	mu       sync.Mutex       //更小的锁，保护下面的变量
	seq      uint64           //请求编号
	pending  map[uint64]*Call //存储未处理完的请求,建是编号，value是Call实例
	closing  bool             //用户主动关闭
	shutdown bool             //服务器告知关闭(一般是发生错误是关闭)
}

var ErrShutdown = errors.New("connection is shut down")

// 关闭连接
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing { //已关闭
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

//判断客户端是否继续正常工作

func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.closing && !client.shutdown
}

// 注册Call
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

// 移除Call
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()

	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// 服务器或客户端发生错误时调用，将shutdown设置为true,且将错误信息通知所有peding状态的call
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()

	client.shutdown = true

	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

// 接收响应
func (client *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		case call == nil:
			//部分写入或者是已经移除
			err = client.cc.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body" + err.Error())
			}
			call.done()
		}
	}

	//发生错误
	client.terminateCalls(err)
}

func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodcType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodcType)
		log.Println("rpc client:codec error:", err)
		return nil, err
	}

	//send options(json) to server
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client:options error:", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *Option) *Client {
	client := &Client{
		seq:     1,
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

// 此处可变参的目的是，可以传一个opt,也可以不传采用default的opt
func parseOption(opts ...*Option) (*Option, error) {
	//if opts is nil or pass nil as parameter
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}

	if len(opts) != 1 {
		return nil, errors.New("number of option is more than 1")
	}
	opt := opts[0]

	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodcType == "" {
		opt.CodcType = DefaultOption.CodcType
	}
	return opt, nil
}

// // 连接指定RPC服务器
// func Dial(network, address string, opts ...*Option) (client *Client, err error) {
// 	opt, err := parseOption(opts...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	conn, err := net.Dial(network, address)
// 	if err != nil {
// 		return nil, err
// 	}

// 	//也就是NewClient调用失败
// 	defer func() {
// 		if client == nil {
// 			_ = conn.Close()
// 		}
// 	}()
// 	return NewClient(conn, opt)
// }

func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	//注册call
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	//准备header
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	//编码和发送
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)

		//write失败，客户端已经开始处理
		if call != nil {
			call.Error = err
			call.done()
		}
	}

}

// 异步接口
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10) //这里1也可以，官方库实现中是10
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)

	//这里其实可以直接go client.send(call)
	//因为不需要等待client.send
	return call

}

// 同步接口
func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}


//用来存放 NewClient的返回结果
type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *Option) (client *Client, err error)



//对代码是略微进行重构，加一层中间件，用来处理超时
func dialTimeout(f newClientFunc, network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOption(opts...)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout(network, address, opt.ConnectTimeout)

	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()

	ch := make(chan clientResult)


	//让子协程去跑函数f(NewClient),将返回值写入管道
	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{client: client, err: err}
	}()

	//如果为0,说明不限时间，就直接阻塞等待管道的返回结果
	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}

	//select字段，超时返回err,不超时返回结果
	select {
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}


func Dial(network,address string ,opts...*Option)(*Client,error) {
	return dialTimeout(NewClient,network,address,opts...)
}