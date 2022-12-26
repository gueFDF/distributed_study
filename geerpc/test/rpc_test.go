package test

import (
	"encoding/json"
	"fmt"
	"geerpc"
	"geerpc/codec"
	"log"
	"net"
	"reflect"
	"testing"
)

func TestServer(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:8888")
	if err != nil {
		println("Listen is err:", err)
		return
	}

	server := geerpc.NewServer()

	server.Accept(l)
}

func TestClient(t *testing.T) {
	conn, _ := net.Dial("tcp", "127.0.0.1:8888")
	defer func() { _ = conn.Close() }()

	json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	cc := codec.NewGobCodec(conn)

	for i := 0; i < 100; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		cc.Write(h, fmt.Sprintf("geerpc req %d", h.Seq))
		cc.ReadHeader(h)
		var reply string
		cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}

// 同步接口测试
func TestClient_sync(t *testing.T) {
	client, _ := geerpc.Dial("tcp", "127.0.0.1:8888")

	defer func() { client.Close() }()

	for i := 0; i < 5; i++ {
		args := fmt.Sprintf("geerpc req %d", i)
		var reply string
		if err := client.Call("Foo.sum", args, &reply); err != nil {
			log.Fatal("call Foo.Sum error:", err)
		}
		println("reply:", reply)
	}
}

// 异步接口测试
func TestClient_async(t *testing.T) {
	client, _ := geerpc.Dial("tcp", "127.0.0.1:8888")
	defer func() { client.Close() }()
	done := make(chan *geerpc.Call, 10)
	for i := 0; i < 5; i++ {
		args := fmt.Sprintf("geerpc req %d", i)
		var reply string
		client.Go("Foo.sum", args, &reply, done)
	}

	var temp *geerpc.Call
	for i := 0; i < 5; i++ {
		select {
		case temp = <-done:
			fmt.Println("reply:", reflect.ValueOf(temp.Reply).Elem())
		}
	}

}
