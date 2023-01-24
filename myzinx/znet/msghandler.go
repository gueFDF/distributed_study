package znet

import (
	"log"
	"myzinx/ziface"
	"strconv"
)

type MsgHandle struct {
	Apis map[uint32]ziface.IRouter
}

func NewMsgHandle() *MsgHandle {
	return &MsgHandle{Apis: make(map[uint32]ziface.IRouter)}
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
