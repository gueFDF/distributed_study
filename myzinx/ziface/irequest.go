// 消息请求抽象类
package ziface

type IRequest interface {
	GetConnection() IConnection //获取请求连接
	GetData() []byte            //获取请求数据
	GetMsgId() uint32           //获取消息ID
}
