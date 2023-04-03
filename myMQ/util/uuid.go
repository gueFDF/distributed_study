package util

import (
	"crypto/rand"
	"io"
	"log"
)

var UuidChan = make(chan []byte, 1000)

// 不断的制造uuid
func UuidFactory() {
	for {
		UuidChan <- uuid()
	}
}

//输出UUID
func uuid() []byte {
	b := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		log.Fatal(err)
	}
	return b
}
