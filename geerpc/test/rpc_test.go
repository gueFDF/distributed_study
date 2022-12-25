package test

import (
	"encoding/json"
	"fmt"
	"geerpc"
	"geerpc/codec"
	"log"
	"net"
	"testing"
)

func TestServer(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:9999")
	if err != nil {
		println("Listen is err:", err)
		return
	}

	server := geerpc.NewServer()

	server.Accept(l)
}

func TestClient(t *testing.T) {
	conn, _ := net.Dial("tcp", "127.0.0.1:9999")
	defer func() { _ = conn.Close() }()

	json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	cc:=codec.NewGobCodec(conn)

	for i:=0;i<5;i++ {
		h:=&codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq: uint64(i),
		}
		cc.Write(h,fmt.Sprintf("geerpc req %d",h.Seq))
		cc.ReadHeader(h)
		var reply string
		cc.ReadBody(&reply)
		log.Println("reply:",reply)
	}
}
