package znet

import (
	"log"
	"myzinx/utils"
	"myzinx/ziface"
	"strconv"
)

type MsgHandle struct {
	Apis             map[uint32]ziface.IRouter //存放所有处理方法
	WorkerPoolSize   uint32                    //worker数量
	TaskQueue        []chan ziface.IRequest    //任务队列
	IsopenWorkerPool bool                      //是否打开协程池
}

func NewMsgHandle() *MsgHandle {
	return &MsgHandle{
		Apis:             make(map[uint32]ziface.IRouter),
		WorkerPoolSize:   utils.GlobalObject.WorkerPoolSize,
		TaskQueue:        make([]chan ziface.IRequest, utils.GlobalObject.WorkerPoolSize),
		IsopenWorkerPool: false,
	}
}

//以非阻塞方式处理消息

func (mh *MsgHandle) DoMsgHandle(request ziface.IRequest) {
	handler, ok := mh.Apis[request.GetMsgId()]
	if !ok {
		log.Println("api msgId = ", request.GetMsgId(), " is not FOUND!")
		return
	}

	//执行对应处理方式
	handler.PreHandle(request)
	handler.Handle(request)
	handler.PostHandle(request)

}

// 为消息添加具体处理逻辑
func (mh *MsgHandle) AddRouter(msgID uint32, router ziface.IRouter) {
	//判断方法是否存在
	if _, ok := mh.Apis[msgID]; ok {
		panic("repeated api , msgId = " + strconv.Itoa(int(msgID)))
	}

	//添加msg与api的绑定关系
	mh.Apis[msgID] = router
	log.Println("Add api msgid = ", msgID)
}

// 启动一个Worker工作流程
func (mh *MsgHandle) StartOneWorker(workerID int, taskQueue chan ziface.IRequest) {
	log.Println("Worker ID = ", workerID, " is started.")
	//不断等待队列中的消息
	for {
		select {
		//有消息则取出队列中的Request ,并执行绑定的业务方法
		case request := <-taskQueue:
			mh.DoMsgHandle(request)
		}
	}
}

// 启动worker工作池
func (mh *MsgHandle) StartWorkerPool() {
	if mh.IsopenWorkerPool {
		log.Println("协程池已经打开，无需重复打开")
		return
	}
	mh.IsopenWorkerPool = true
}

// 分发任务
func (mh *MsgHandle) SendMsgToTaskQueue(request ziface.IRequest) {
	//根据ID来分配当前连接应该由哪一个Worker负责处理
	//轮询的分配法则

	//得到需要处理此条连接的workerID
	workerID := request.GetConnection().GetConnID() % mh.WorkerPoolSize
	log.Println("Add ConnID=", request.GetConnection().GetConnID(), " request msgID=", request.GetMsgId(), "to workerID=", workerID)
	if mh.TaskQueue[workerID] == nil {
		mh.TaskQueue[workerID] = make(chan ziface.IRequest, utils.GlobalObject.MaxWorkerTaskLen)
		go mh.StartOneWorker(int(workerID), mh.TaskQueue[workerID])
	}
	//将请求任务发送给任务队列
	mh.TaskQueue[workerID] <- request
}

func (mh *MsgHandle) IsOpen() bool {
	return mh.IsopenWorkerPool
}
