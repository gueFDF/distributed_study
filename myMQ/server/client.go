package server

import (
	"encoding/binary"
	"io"
	"log"
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

func (c *Client) Getstate() int {
	return c.stat
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
	log.Printf("CLIENT(%s): closing", c.String())
	c.conn.Close()
}
