# 消息编码

使用 encoding/gob 实现消息的编解码(序列化与反序列化)

将请求和响应中的参数和返回值抽象为body
将服务信息如服务名，方法名等其他信息存放在Header中

```go
type Header struct {
	ServiceMethod string
	Seq           uint64  //requse ID
	Error         string
}
```

* ```ServiceMethod``` 是服务名和方法名，通常与 Go 语言中的结构体和方法相映射。
* ```Seq``` 是请求的序号，也可以认为是某个请求的 ID，用来区分不同的请求。
* ```Error``` 是错误信息，客户端置为空，服务端如果如果发生错误，将错误信息置于 Error 中


消息的decode和encode的方式有很多，例如json,xml,protobuf 包括go特有的gob,所要实现的框架需要可以做到让用户自主选择编码方式。
为了实现不同的Codec实例，先对消息体编解码的接口进行抽象，抽象出Codec
```go
type Codec interface {
	io.Closer
	ReadHeader(*Header) error                          
	ReadBody(interface{})error
	Write(*Header,interface{}) error
}
```

将编码类型枚举出来，（此处只列出了god和josn）
```go
//编码类型
type Type string 

const (
	GobType Type ="application/god"
	JsonType Type ="application/json" //未实现
)
```
因为不同的编码类型对应的函数方法也是不同的，为了方便管理，拿go中的map进行管理，在init函数中为不同编码类型注册对应的方法
```go
var NewCodecFuncMap map[Type]NewCodecFunc

func init(){
	NewCodecFuncMap=make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType]=NewGobCodec //为对应类型注册方法
}
```


# 具体实现
因为不同类型的编码方式，写法大致相同，所以在文章中只实现了gob一种

首先需要定义出GobCodec实例
```go
type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}
```
* conn 通常是通过TCP建立socket时得到的连接实例，由构建函数传入
* buf 一个带缓冲的Writer,防止阻塞
* dec gob的Decoder实例，将从conn中读取的信息进行解码
* enc gob的Encoder实例，将信息进行编码，写到conn中
(此处笔者其实有一个疑问，为什么只加入写缓冲，不加入读缓冲，加入读缓冲是否也能带来性能的提升)


var _ Codec = (*GobCodec)(nil) 作用其实就是检查GobCodec是否将Codec中的方法都全部实现


接下来就是Codec中方法的具体实现
```go
func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c*GobCodec)Write(h*Header,body interface{})(err error) {
	defer func(){
		_=c.buf.Flush() //刷新写缓冲区
		if err!=nil {
			_=c.Close()
		}
	}()

	if err:=c.enc.Encode(h);err!=nil {
		log.Println("rpc codec:gob error edcoding header:",err)
		return err
	}
	if err:=c.enc.Encode(body);err!=nil {
		log.Panicln("rpc codec:god error encoding body:",err)
		return err
	}
	return nil
}



func (c*GobCodec)Close()error {
	return c.conn.Close()
}
```




# server部分
客户端与服务端的通信需要协商一些内容，例如 HTTP 报文，分为 header 和 body 2 部分，body 的格式和长度通过 header 中的 Content-Type 和 Content-Length 指定，服务端通过解析 header 就能够知道如何从 body 中读取需要的信息。对于 RPC 协议来说，这部分协商是需要自主设计的。为了提升性能，一般在报文的最开始会规划固定的字节，来协商相关的信息。比如第1个字节用来表示序列化方式，第2个字节表示压缩方式，第3-6字节表示 header 的长度，7-10 字节表示 body 的长度。


对于 GeeRPC 来说，目前需要协商的唯一一项内容是消息的编解码方式。我们将这部分信息，放到结构体 Option 中承载。

```go
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
```

为了简单，GeeRPC实现中，option固定采用json形式进行编码，后续的header和body的编码方式有option中的CodeType指定

```
| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
```

服务端具体实现

```go
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
```
for循环等待连接建立，将建立好的连接交给子协程处理，处理过程在ServerConn
先用json解码获取option,然后个根据opt.CodcType，解码后续的header和body
```go
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
```


serveCodec 的过程非常简单。主要包含三个阶段

* 读取请求 readRequest
* 处理请求 handleRequest
* 回复请求 sendResponse
之前提到过，在一次连接中，允许接收多个请求，即多个 request header 和 request body，因此这里使用了 for 无限制地等待请求的到来，直到发生错误（例如连接被关闭，接收到的报文有问题等），这里需要注意的点有三个：

handleRequest 使用了协程并发执行请求。
处理请求是并发的，但是回复请求的报文必须是逐个发送的，并发容易导致多个回复报文交织在一起，客户端无法解析。在这里使用锁(sending)保证。
尽力而为，只有在 header 解析失败时，才终止循环。

```go
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
```

存放一次请求的所有信息
```go
type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}
```

readRequest
```go
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
```


sendResponse
```go
func (server*Server)sendResponse(cc codec.Codec,h*codec.Header,body interface{},sending*sync.Mutex) {
	//因为客户端只是一个，为了防止客户端数据接收混乱，所以需要加锁
	sending.Lock()
	defer sending.Unlock()
	if err:=cc.Write(h,body);err!=nil {
		log.Println("rpc server:write response error:",err)
	}
}
```

handleRequest
```go
func(server*Server)handleRequest(cc codec.Codec,req*request,sending*sync.Mutex,wg*sync.WaitGroup){
	//应根据请求类型执行相应的Rpc方法（目前暂未实现服务注册，打印一句话来代替）

	defer wg.Done()
	log.Println(req.h,req.argv.Elem())

	req.replyv=reflect.ValueOf(fmt.Sprintf("geerpc resp %d",&req.h.Seq))
	server.sendResponse(cc,req.h,req.replyv.Interface(),sending)
}
```