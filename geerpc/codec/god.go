package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// 实例
type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}


//使用匿名变量，作用是检查GobCodec是否将Codec中的所有接口都是实现
var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	//此处有一个疑问，为什么只加写缓冲，不加读缓冲
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}



func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c*GobCodec)Write(h*Header,body interface{})(err error) {
	defer func(){
		_=c.buf.Flush() //刷新写缓冲区
		if err!=nil {
			_=c.Close()
		}
	}()

	if err:=c.enc.Encode(h);err!=nil {
		log.Println("rpc codec:gob error edcoding header:",err)
		return err
	}
	if err:=c.enc.Encode(body);err!=nil {
		log.Panicln("rpc codec:god error encoding body:",err)
		return err
	}
	return nil
}



func (c*GobCodec)Close()error {
	return c.conn.Close()
}