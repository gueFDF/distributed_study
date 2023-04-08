package util

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
)

var UuidChan = make(chan []byte, 1000)

// 不断的制造uuid
func UuidFactory(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case UuidChan <- uuid():
		}
	}
}

// 输出UUID
func uuid() []byte {
	b := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

// 将UUID转化为一个字符串
func UuidToStr(b []byte) string {

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
