package message

import (
	"errors"
	"log"
	"myMQ/util"
	"time"
)

// 解耦和
type Consumer interface {
	Close()
}

type Channel struct {
	name                string
	addClientChan       chan util.ChanReq
	removeClientChan    chan util.ChanReq
	clients             []Consumer        //管理所有的client
	incomingMessageChan chan *Message     //接收生产者的消息
	msgChan             chan *Message     //暂存消息
	clientMessageChan   chan *Message     //消息会被发送到这个管道，后续有消费者使用
	exitChan            chan util.ChanReq //用来管道关闭

	inFilghtMessageChan chan *Message       //暂时存放发送中的消息
	inFilghtMessages    map[string]*Message //管理发送中的消息

	finishMessage      chan util.ChanReq //存放发送成功的message的信息
	requeueMessageChan chan util.ChanReq //要重新发送的消息

}

// 推送消息
func (c *Channel) PutMessage(msg *Message) {
	c.incomingMessageChan <- msg
}

// 拉取消息
func (c *Channel) PullMessage() *Message {
	return <-c.clientMessageChan
}

// 添加客户端
func (c *Channel) addClient(client Consumer) {
	log.Printf("Channl(%s): adding client...", c.name)
	doneChan := make(chan interface{})
	c.addClientChan <- util.ChanReq{
		Variable: client,
		RetChan:  doneChan,
	}
	<-doneChan
}

// 移除客户
func (c *Channel) RemoveClient(client Consumer) {
	log.Printf("Channel(%s): remove client...", c.name)
	doneChan := make(chan interface{})
	c.removeClientChan <- util.ChanReq{
		Variable: client,
		RetChan:  doneChan,
	}
	<-doneChan
}

// 不停的将消息从msgChan中的读取，写入clientMessageChan管道中
func (c *Channel) MessagePump(closechan chan struct{}) {
	var msg *Message
	for {
		select {
		case msg = <-c.msgChan:
		case <-closechan:
			return
		}
		if msg != nil {
			c.inFilghtMessageChan <- msg //将发送中的消息加入管道
		}
		c.clientMessageChan <- msg
	}
}

// 关闭管道
func (c *Channel) Close() error {
	errChan := make(chan interface{})
	c.exitChan <- util.ChanReq{
		RetChan: errChan,
	}

	err, _ := (<-errChan).(error)
	return err
}

// 保存发送中的消息
func (c *Channel) pushInFilghtMessage(msg *Message) {
	c.inFilghtMessages[util.UuidToStr(msg.Uuid())] = msg
}

// 删除发送中的消息
func (c *Channel) popInFilghtMessage(uuidStr string) (*Message, error) {
	//确保消息存在
	msg, ok := c.inFilghtMessages[uuidStr]
	if !ok {
		return nil, errors.New("uuid not in flight")
	}
	delete(c.inFilghtMessages, uuidStr)
	msg.EndTimer()
	return msg, nil
}

func (c *Channel) RequeueRouter(closeChan chan struct{}) {
	for {
		select {
		case msg := <-c.inFilghtMessageChan: // 将暂存发送中消息的管道的消息放到map
			c.pushInFilghtMessage(msg)
			go func(msg *Message) { //处理超时
				select {
				case <-time.After(60 * time.Second):
					log.Printf("CHANNEL(%s): auto requeue of message(%s)", c.name, util.UuidToStr(msg.Uuid()))
				case <-msg.timeout:
					return
				}
				err := c.RequeueMessage(util.UuidToStr(msg.Uuid()))
				if err != nil {
					log.Printf("ERROR: channel(%s) - %s", c.name, err.Error())
				}
			}(msg)
		case requeueReq := <-c.requeueMessageChan: //将要重新发送消息管道中的消息重新发送
			uuidStr := requeueReq.Variable.(string)
			msg, err := c.popInFilghtMessage(uuidStr)
			if err != nil {
				log.Printf("ERROR: failed to requeue message(%s) - %s", uuidStr, err.Error())
			} else {
				go func(msg *Message) {
					c.PutMessage(msg)
				}(msg)
			}
			requeueReq.RetChan <- err
		case finishReq := <-c.finishMessage: //消息完成发送，从map中将消息删除
			uuidStr := finishReq.Variable.(string)
			_, err := c.popInFilghtMessage(uuidStr)
			if err != nil {
				log.Printf("ERROR: failed to finish message(%s) - %s", uuidStr, err.Error())
			}
			finishReq.RetChan <- err
		case <-closeChan:
			return
		}
	}
}

func (c *Channel) RequeueMessage(uuidStr string) error {
	errChan := make(chan interface{})
	c.requeueMessageChan <- util.ChanReq{
		Variable: uuidStr,
		RetChan:  errChan,
	}
	err, _ := (<-errChan).(error)
	return err
}

// 消息成功发送
func (c *Channel) FinishMessage(uuidStr string) error {
	errChan := make(chan interface{})
	c.finishMessage <- util.ChanReq{
		Variable: uuidStr,
		RetChan:  errChan,
	}
	err, _ := (<-errChan).(error)

	return err
}

// 路由（事件处理）
func (c *Channel) Router() {
	var (
		clientReq util.ChanReq
		closeChan = make(chan struct{}) //目的是通知MessagePump协程关闭
	)
	go c.MessagePump(closeChan)
	go c.RequeueRouter(closeChan)
	for {
		select {
		case clientReq = <-c.addClientChan:
			client := clientReq.Variable.(Consumer)
			c.clients = append(c.clients, client)
			log.Printf("CHANNEL(%s) added client %#v", c.name, client)
			clientReq.RetChan <- struct{}{}
		case clientReq = <-c.removeClientChan:
			client := clientReq.Variable.(Consumer)
			indexToRemove := -1
			for k, v := range c.clients {
				if v == client {
					indexToRemove = k
					break
				}
			}
			if indexToRemove == -1 {
				log.Printf("ERROR: could not find client(%#v) in clients(%#v)", client, c.clients)
			} else {
				c.clients = append(c.clients[:indexToRemove], c.clients[indexToRemove+1:]...)
				log.Printf("CHANNEL(%s) removed client %#v", c.name, client)
			}
		case msg := <-c.incomingMessageChan:
			select {
			// 防止因 msgChan 缓冲填满时造成阻塞，加上一个 default 分支直接丢弃消息
			case c.msgChan <- msg:
				log.Printf("CHANNEL(%s) wrote message", c.name)
			default:
			}

		case closeReq := <-c.exitChan:
			log.Printf("CHANNEL(%s) is closing", c.name)
			close(closeChan)
			for _, consumer := range c.clients {
				consumer.Close() //告知MessagePump协程退出
			}
			closeReq.RetChan <- nil
		}
	}
}
