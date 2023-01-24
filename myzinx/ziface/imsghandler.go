package ziface

//消息管理模块

type IMsgHandle interface {
	DoMsgHandle(request IRequest)  //以非阻塞方式处理消息
	AddRouter(msdId uint32,router IRouter)  //为消息添加具体的处理逻辑
}
