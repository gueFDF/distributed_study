package codec

import "io"

type Header struct {
	ServiceMethod string
	Seq           uint64  //requse ID
	Error         string
}


//进一步抽象
type Codec interface {
	io.Closer
	ReadHeader(*Header) error                          
	ReadBody(interface{})error
	Write(*Header,interface{}) error
}


     


type NewCodecFunc func(io.ReadWriteCloser) Codec


//编码类型
type Type string 

const (
	GobType Type ="application/god"
	JsonType Type ="application/json" //未实现
)


//type-method
var NewCodecFuncMap map[Type]NewCodecFunc

func init(){
	NewCodecFuncMap=make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType]=NewGobCodec //为对应类型注册方法
}

