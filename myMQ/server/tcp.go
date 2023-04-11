package server

import (
	"context"
	"myMQ/logs"
	"net"
)

func TcpServer(ctx context.Context, addr, port string) {
	fqAddress := addr + ":" + port
	listener, err := net.Listen("tcp", fqAddress)
	if err != nil {
		panic("tcp listen(" + fqAddress + ") failed")
	}
	logs.Info("listening for clients on %s", fqAddress)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				panic("accept failed: " + err.Error())
			}
			client := NewClient(conn, conn.RemoteAddr().String())
			go client.Handle(ctx)
		}
	}
}
