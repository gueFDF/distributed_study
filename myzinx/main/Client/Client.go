package main

import (
	"fmt"
	"myzinx/znet"
	"net"
	"time"
)

/*
模拟客户端
*/
func main() {

	fmt.Println("Client Test ... start")
	//3秒之后发起测试请求，给服务端开启服务的机会
	time.Sleep(3 * time.Second)

	conn, err := net.Dial("tcp", "127.0.0.1:7777")
	if err != nil {
		fmt.Println("client start err, exit!")
		return
	}

	for {
		//发封包message消息
		dp := znet.NewDataPack()
		msg, _ := dp.Pack(znet.NewMsgPackage(0, []byte("Zinx V0.5 Client Test Message")))
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Println("write error err ", err)
			return
		}

		//将headData字节流 拆包到msg中
		msgHead, err := dp.Unpack(conn)
		if err != nil {
			fmt.Println("server unpack err:", err)
			return
		}

		if msgHead.GetDataLen() > 0 {

			fmt.Println("==> Recv Msg: ID=",msgHead.GetMsgId(), ", len=",msgHead.GetDataLen(), ", data=", string(msgHead.GetData()))
		}

		time.Sleep(1 * time.Second)
	}
}
