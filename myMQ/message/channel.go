package message

import (
	"log"
	"myMQ/util"
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

// 路由（事件处理）
func (c *Channel) Router() {
	var (
		clientReq util.ChanReq
		closeChan = make(chan struct{}) //目的是通知MessagePump协程关闭
	)
	go c.MessagePump(closeChan)
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

// 不停的将消息从msgChan中的读取，写入clientMessageChan管道中
func (c *Channel) MessagePump(closechan chan struct{}) {
	var msg *Message
	for {
		select {
		case msg = <-c.msgChan:
			c.clientMessageChan <- msg
		case <-closechan:
			return
		}

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
