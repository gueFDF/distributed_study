package message

//前16位为消息的唯一标识

type Message struct {
	data    []byte
	timeout chan struct{}
}

func NewMessage(data []byte) *Message {
	return &Message{
		data: data,
	}
}

func (m *Message) Uuid() []byte {
	return m.data[:16]
}

func (m *Message) Body() []byte {
	return m.data[16:]
}

func (m *Message) Data() []byte {
	return m.data
}

// 用来结束超时处理协程
func (m *Message) EndTimer() {
	select {
	case m.timeout <- struct{}{}:
	default:
	}
}
