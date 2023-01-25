package ziface

//消息管理模块

type IMsgHandle interface {
	DoMsgHandle(request IRequest)           //以非阻塞方式处理消息
	AddRouter(msdId uint32, router IRouter) //为消息添加具体的处理逻辑
	StartWorkerPool()                       //启动工作池
	SendMsgToTaskQueue(request IRequest)    //将消息交给TaskQueue,由worker进行处理

	IsOpen() bool  //是否打开协程池
}
