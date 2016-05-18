package main

import (
	"fmt"
	"github.com/autogrowsystems/wray"
	"time"
)

func main() {
	wray.RegisterTransports([]wray.Transport{&wray.HTTPTransport{}})
	client := wray.NewFayeClient("http://localhost:5000/faye")
	client.AddExtension(client) // log all packets in/out
	go client.Listen()

	fmt.Print("****** SIMPLE SUBSCRIBE EXAMPLE ******\n\n")

	go func() {
		for {
			client.Publish("/foo", map[string]interface{}{"hello": "from foo"})
			time.Sleep(2 * time.Second)
		}
	}()

	msgChan, _ := client.Subscribe("/foo")
	for {
		msg := <-msgChan
		fmt.Println("-------------------------------------------")
		fmt.Println(msg.Data())
	}
}
