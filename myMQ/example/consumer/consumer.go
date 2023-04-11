package main

import (
	"myMQ/client.go"
	"myMQ/logs"

	"myMQ/util"
)

func main() {
	consumeClient := client.NewClient(nil)
	err := consumeClient.Connect("10.30.0.192", 5151)
	if err != nil {
		logs.Fatal(err)
	}
	consumeClient.WriteCommand(consumeClient.Subscribe("test", "ch"))

	for {
		msg, err := consumeClient.ReadResponse()
		if err != nil {
			logs.Error(err.Error())
		}
		logs.Info("%s - %s", util.UuidToStr(msg.Uuid()), msg.Body())

		err2 := consumeClient.WriteCommand(consumeClient.Finish(util.UuidToStr(msg.Uuid())))
		if err2 != nil {
			logs.Error("finish err: ", err)
		}
	}
}
