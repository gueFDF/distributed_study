package protocol

import (
	"bufio"
	"log"
	"myMQ/message"
	"myMQ/util"
	"reflect"
	"strings"
)

//实现四种协议，SUB(订阅)，GET(读取),FIN(完成)，REQ(重入)

type Protocol struct {
	channel *message.Channel
}

func (p *Protocol) IOLoop(client StatefulReadWriter) error {
	var (
		err  error
		line string
		resp []byte
	)
	client.SetState(ClientInit)

	reader := bufio.NewReader(client)

	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}
		//将"\n"替换为""
		line = strings.Replace(line, "\n", "", -1)
		//将"\r"替换为""
		line = strings.Replace(line, "\r", "", -1)
		params := strings.Split(line, " ")

		log.Printf("PROTOCOL: %#v", params)

		resp, err = p.Execute(client, params...)

		if err != nil {
			_, err = client.Write([]byte(err.Error()))
			if err != nil {
				break
			}
			continue
		}

		if resp != nil {
			_, err = client.Write(resp)
			if err != nil {
				break
			}
		}
	}
	return err
}

func (p *Protocol) Execute(client StatefulReadWriter, params ...string) ([]byte, error) {
	var (
		err  error
		resp []byte
	)

	typ := reflect.TypeOf(p)
	args := make([]reflect.Value, 3)
	args[0] = reflect.ValueOf(p)
	args[1] = reflect.ValueOf(client)
	cmd := strings.ToUpper(params[0])
	if method, ok := typ.MethodByName(cmd); ok {
		args[2] = reflect.ValueOf(params)
		returnValues := method.Func.Call(args)

		if !returnValues[0].IsNil() {
			resp = returnValues[0].Interface().([]byte)
		}

		if !returnValues[1].IsNil() {
			err = returnValues[1].Interface().(error)
		}
		return resp, err
	}
	return nil, ClientErrInvalid
}

// 绑定
func (p *Protocol) SUB(client StatefulReadWriter, params []string) ([]byte, error) {
	if client.GetState() != ClientInit {
		return nil, ClientErrInvalid
	}

	if len(params) < 3 {
		return nil, ClientErrInvalid
	}

	topicName := params[1]
	if len(topicName) == 0 {
		return nil, ClientErrBadTopic
	}

	channelName := params[2]
	if len(channelName) == 0 {
		return nil, ClientErrBadChannel
	}

	client.SetState(ClientWaitGet)
	topic := message.GetTopic(topicName)
	p.channel = topic.GetChannel(channelName)

	return nil, nil
}

//向绑定的channel发送消息

func (p *Protocol) GET(client StatefulReadWriter, params []string) ([]byte, error) {
	if client.GetState() != ClientWaitGet {
		return nil, ClientErrInvalid
	}

	msg := p.channel.PullMessage()

	if msg == nil {
		log.Printf("ERROR: msg == nil")
		return nil, ClientErrBadMessage
	}

	uuidStr := util.UuidToStr(msg.Uuid())
	log.Printf("PROTOCOL: writing msg(%s) to client(%s) - %s", uuidStr, client.String(), string(msg.Body()))
	client.SetState(ClientWaitResponse)

	return msg.Data(), nil
}

// 接收到消息
func (p *Protocol) FIN(client StatefulReadWriter, params []string) ([]byte, error) {
	if client.GetState() != ClientWaitResponse {
		return nil, ClientErrInvalid
	}

	if len(params) < 2 {
		return nil, ClientErrInvalid
	}

	uuidStr := params[1]
	err := p.channel.FinishMessage(uuidStr)
	if err != nil {
		return nil, err
	}

	client.SetState(ClientWaitGet)

	return nil, nil
}

// 重发
func (p *Protocol) REQ(client StatefulReadWriter, params []string) ([]byte, error) {
	if client.GetState() != ClientWaitResponse {
		return nil, ClientErrInvalid
	}

	if len(params) < 2 {
		return nil, ClientErrInvalid
	}

	uuidStr := params[1]
	err := p.channel.RequeueMessage(uuidStr)
	if err != nil {
		return nil, err
	}

	client.SetState(ClientWaitGet)

	return nil, nil

}
