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
	var foo Foo

	l, err := net.Listen("tcp", "127.0.0.1:8888")
	if err != nil {
		println("Listen is err:", err)
		return
	}

	server := geerpc.NewServer()

	if err := server.Register(&foo); err != nil {
		log.Fatal("register error", err)
	}
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

// TODO :day2 支持异步和并发的高性能客户端
// 同步接口测试
func BenchmarkClient_sync(b *testing.B) {
	client, _ := geerpc.Dial("tcp", "127.0.0.1:8888")

	defer func() { client.Close() }()

	for i := 0; i < 5; i++ {
		args := fmt.Sprintf("geerpc req %d", i)
		var reply string
		if err := client.Call("Foo.Sum", args, &reply); err != nil {
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

// TODO :完成服务注册
type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func TestService(t *testing.T) {
	client, _ := geerpc.Dial("tcp", "127.0.0.1:8888")
	defer func() { _ = client.Close() }()
	for i := 0; i < 10; i++ {
		args := &Args{i, i * i}
		var reply int
		if err := client.Call("Foo.Sum", args, &reply); err != nil {
			log.Fatal("call Foo.Sum error:", err)
		}
		log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
	}
}
