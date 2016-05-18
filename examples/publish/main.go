package main

import (
	"fmt"
	"github.com/autogrowsystems/wray"
	// "gopkg.in/autogrowsystems/wray.v1"
	"time"
)

func main() {
	wray.RegisterTransports([]wray.Transport{&wray.HTTPTransport{}})
	// wray.RegisterTransports([]wray.Transport{&wray.HttpTransport{}})
	client := wray.NewFayeClient("http://localhost:5000/faye")
	client.AddExtension(client) // log all packets in/out

	go client.Listen()

	fmt.Print("****** PUBLISH EXAMPLE ******\n\n")

	params := map[string]interface{}{"hello": "from golang"}

	for {
		err := client.Publish("/foo", params)
		if err != nil {
			fmt.Printf("\n[ERROR] %s\n\n", err)
		}

		time.Sleep(2 * time.Second)
	}
}
