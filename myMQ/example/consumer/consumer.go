package main

import (
	"log"
	"myMQ/client.go"
	"myMQ/util"
)

func main() {
	consumeClient := client.NewClient(nil)
	err := consumeClient.Connect("127.0.0.1", 5151)
	if err != nil {
		log.Fatal(err)
	}
	consumeClient.WriteCommand(consumeClient.Subscribe("test", "ch"))

	for {
		msg, err := consumeClient.ReadResponse()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s - %s", util.UuidToStr(msg.Uuid()), msg.Body())
		consumeClient.WriteCommand(consumeClient.Finish(util.UuidToStr(msg.Uuid())))
	}
}
