package server

import (
	"context"
	"encoding/binary"
	"io"
	"myMQ/logs"
	"myMQ/protocol"
)

type Client struct {
	conn io.ReadWriteCloser
	name string
	stat int
}

func NewClient(conn io.ReadWriteCloser, name string) *Client {
	return &Client{
		conn: conn,
		name: name,
		stat: -1,
	}
}

func (c *Client) String() string {
	return c.name
}

func (c *Client) GetState() int {
	return c.stat
}

func (c *Client) SetState(state int) {
	c.stat = state
}

func (c *Client) Read(data []byte) (int, error) {
	return c.conn.Read(data)
}

func (c *Client) Write(data []byte) (int, error) {
	var err error
	//处理黏包
	err = binary.Write(c.conn, binary.BigEndian, int32(len(data)))
	if err != nil {
		return 0, err
	}
	n, err := c.conn.Write(data)
	if err != nil {
		return 0, err
	}
	return n + 4, nil
}

func (c *Client) Close() {
	logs.Info("CLIENT(%s): closing", c.String())
	c.conn.Close()
}

// 一个client绑定一个protocol
func (c *Client) Handle(ctx context.Context) {
	defer c.Close()

	proto := &protocol.Protocol{}
	err := proto.IOLoop(ctx, c)

	if err != nil {
		logs.Error("ERROR: client(%s) - %s", c.String(), err.Error())
		return
	}

}
