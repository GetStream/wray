package main

import (
	"fmt"
	"github.com/autogrowsystems/wray"
)

type authExtn struct {
	token string
}

// Out sets the auth token on the map in Ext()
func (extn *authExtn) Out(msg wray.Message) {
	ext := msg.Ext()
	ext["authToken"] = extn.token // this change is
}

// In just satisfies the interface
func (extn *authExtn) In(msg wray.Message) {}

type spyExtn struct{}

func (spy *spyExtn) Out(msg wray.Message) {
	fmt.Printf("[SPY] OUTGOING %s: %+v\n", msg.Channel(), msg)
}

func (spy *spyExtn) In(msg wray.Message) {
	fmt.Printf("[SPY] INCOMING %s: %+v\n", msg.Channel(), msg)
}

func main() {
	// e := &authExtn{"abcdef0123456789"}
	// e2 := &spyExtn{}
	url := "http://localhost:5000/faye"

	wray.RegisterTransports([]wray.Transport{&wray.HTTPTransport{}})
	faye := wray.NewFayeClient(url)
	// faye.AddExtension(e)
	// faye.AddExtension(e2)

	go faye.Listen()

	// when sent this message will have the auth token added to it
	faye.Publish("/test", map[string]interface{}{"hello": "world"})
}
