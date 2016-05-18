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

	chans := []chan wray.Message{
		make(chan wray.Message),
		make(chan wray.Message),
		make(chan wray.Message),
	}

	fmt.Print("****** COMPLEX SUBSCRIBE EXAMPLE ******\n\n")

	go client.WaitSubscribe("/foo", chans[0])
	go client.WaitSubscribe("/bar", chans[1])
	go client.WaitSubscribe("/baz", chans[2])

	go func() {
		for {
			client.Publish("/foo", map[string]interface{}{"hello": "from foo"})
			client.Publish("/bar", map[string]interface{}{"hello": "from bar"})
			client.Publish("/baz", map[string]interface{}{"hello": "from baz"})
			time.Sleep(2 * time.Second)
		}
	}()

	for {
		select {
		case msg := <-chans[0]:
			go printMsg(msg)
		case msg := <-chans[1]:
			go printMsg(msg)
		case msg := <-chans[2]:
			go printMsg(msg)
		}
	}
}

func printMsg(msg wray.Message) {
	fmt.Println("\n-------------------------------------------")
	fmt.Printf("%+v\n", msg.Data())
	fmt.Print("-------------------------------------------\n\n")
}
