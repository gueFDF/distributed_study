package test

import (
	"fmt"
	"myMQ/util"
	"testing"
)

func TestUuid(t *testing.T) {

	go util.UuidFactory()
	uid1 := <-util.UuidChan
	uid2 := <-util.UuidChan
	uid3 := <-util.UuidChan

	fmt.Println(uid1)
	
	println(uid2)
	println(uid3)
	fmt.Printf(string(uid1))

}
