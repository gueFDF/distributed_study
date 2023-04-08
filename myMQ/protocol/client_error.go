package protocol

import "io"

// 定义一些客户端的错误以及需要用到的常量和接口
const (
	ClientInit = iota
	ClientWaitGet
	ClientWaitResponse
)

type StatefulReadWriter interface {
	io.ReadWriter
	GetState() int      //获取状态
	SetState(state int) //设置状态
	String() string
	Close()
}

type ClientError struct {
	errStr string
}

func (e ClientError) Error() string {
	return e.errStr
}

var (
	ClientErrInvalid    = ClientError{"E_INVALID"}
	ClientErrBadTopic   = ClientError{"E_BAD_TOPIC"}
	ClientErrBadChannel = ClientError{"E_BAD_CHANNEL"}
	ClientErrBadMessage = ClientError{"E_BAD_MESSAGE"}
)
