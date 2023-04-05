package message

import (
	"log"
	"myMQ/util"
)

type Topic struct {
	name                string              //name
	newChannelChan      chan util.ChanReq   //新增的管道
	channelMap          map[string]*Channel // 管理所有的channel
	incomingMessageChan chan *Message       //接受消息的管道
	msgChan             chan *Message       //有缓冲，消息的内存队列
	readSyncChan        chan struct{}       //和 routerSyncChan 配合使用保证 channelMap 的并发安全
	routerSyncChan      chan struct{}
	exitChan            chan util.ChanReq //用来接受退出信号
	channelWriteStarted bool              //是否已经向channel发送消息
}

var (
	TopicMap     = make(map[string]*Topic) //管理所有的topic
	newTopicChan = make(chan util.ChanReq)
)

// 创建型的topic
func NewTopic(name string, inMemSize int) *Topic {
	topic := &Topic{
		name:                name,
		newChannelChan:      make(chan util.ChanReq),
		channelMap:          make(map[string]*Channel),
		incomingMessageChan: make(chan *Message),
		msgChan:             make(chan *Message, inMemSize),
		readSyncChan:        make(chan struct{}),
		routerSyncChan:      make(chan struct{}),
		exitChan:            make(chan util.ChanReq),
	}
	//go topic.Router(inMemSize)
	return topic
}

func GetTopic(name string) *Topic {
	topicChan := make(chan interface{})
	newTopicChan <- util.ChanReq{
		Variable: name,
		RetChan:  topicChan,
	}

	return (<-topicChan).(*Topic)
}

func TopicFactory(inMemSize int) {
	var (
		topicReq util.ChanReq
		name     string
		topic    *Topic
		ok       bool
	)
	for {
		topicReq = <-newTopicChan
		name = topicReq.Variable.(string)
		if topic, ok = TopicMap[name]; !ok {
			topic = NewTopic(name, inMemSize)
			TopicMap[name] = topic
			log.Printf("TOPIC %s CREATED", name)
		}
		topicReq.RetChan <- topic
	}
}

// 获取一个channel
func (t *Topic) GetChannel(channelName string) *Channel {
	channelRet := make(chan interface{})
	t.newChannelChan <- util.ChanReq{
		Variable: channelName,
		RetChan:  channelRet,
	}
	return (<-channelRet).(*Channel)
}

// 主要处理逻辑
func (t *Topic) Router(inMemSize int) {
	var (
		msg       *Message
		closeChan chan struct{}
	)
	for {
		select {
		case channelReq := <-t.newChannelChan:
			channelName := channelReq.Variable.(string)
			if channel, ok := t.channelMap[channelName]; !ok {
				channel = NewChannel(channel.name, inMemSize)
				t.channelMap[channelName] = channel
				log.Printf("TOPIC(%s): new channel(%s)", t.name, channel.name)
				channelReq.RetChan <- channel
				if !t.channelWriteStarted {
					go t.MessagePump(closeChan)
					t.channelWriteStarted = true
				}
			}
		case msg = <-t.incomingMessageChan:
			select {
			case t.msgChan <- msg:
				log.Printf("TOPIC(%s) wrote message", t.name)
			default:
			}
		case <-t.readSyncChan:
			<-t.routerSyncChan
		case closeReq := <-t.exitChan:
			log.Printf("TOPIC(%s): closing", t.name)

			for _, channel := range t.channelMap {
				err := channel.Close()
				if err != nil {
					log.Printf("ERROR: channel(%s) close - %s", channel.name, err.Error())
				}
			}

			close(closeChan)
			closeReq.RetChan <- nil

		}
	}
}

// 发送消息
func (t *Topic) PutMessage(msg *Message) {
	t.incomingMessageChan <- msg
}

func (t *Topic) MessagePump(closechan chan struct{}) {
	var msg *Message
	for {
		select {
		case msg = <-t.msgChan:
		case <-closechan:
			return
		}

		t.readSyncChan <- struct{}{}

		for _, channel := range t.channelMap {
			go func(ch *Channel) {
				ch.PutMessage(msg)
			}(channel)
		}

		t.routerSyncChan <- struct{}{}
	}
}

func (t *Topic) Close() error {
	errChan := make(chan interface{})
	t.exitChan <- util.ChanReq{
		RetChan: errChan,
	}
	err, _ := (<-errChan).(error)
	return err
}
