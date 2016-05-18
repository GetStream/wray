package wray

import (
	"errors"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func Wait(delay time.Duration) {
	time.Sleep(delay)
}

func Pending(description string, execute func()) {
	fmt.Println("PENDING: " + description)
}

func TestInitializingClient(t *testing.T) {
	Convey("puts the client in the unconnected state", t, func() {
		fayeClient := NewFayeClient("http://localhost")
		So(fayeClient.state, ShouldEqual, UNCONNECTED)
		So(fayeClient.schedular, ShouldHaveSameTypeAs, channelSchedular{})
	})
}

func TestSubscribe(t *testing.T) {
	Convey("subscribe to a channel when unconnected", t, func() {
		channel := "/foo/*"

		response := Response(msgWrapper{&message{ID: "1", Channel: "/meta/handshake", Successful: true, ClientID: "client4", SupportedConnectionTypes: []string{"long-polling"}}})

		fakeHTTPTransport := &FakeHTTPTransport{usable: true, response: response}
		registeredTransports = []Transport{fakeHTTPTransport}

		fayeClient := BuildFayeClient().WithTransport(fakeHTTPTransport).Client()

		subscriptionParams := map[string]interface{}{"channel": "/meta/subscribe", "clientId": response.ClientID(), "subscription": "/foo/*", "id": "1"}
		msgChan, err := fayeClient.Subscribe(channel)

		Convey("connects the faye client", func() {
			So(err, ShouldEqual, nil)
			So(fayeClient.state, ShouldEqual, CONNECTED)
		})

		Convey("adds the subscription to the client", func() {
			So(len(fayeClient.subscriptions), ShouldEqual, 1)
			So(fayeClient.subscriptions[0].channel, ShouldEqual, channel)
			So(fayeClient.subscriptions[0].msgChan, ShouldEqual, msgChan)
		})

		Convey("the client send the subscription to the server", func() {
			So(fakeHTTPTransport.sentParams, ShouldResemble, subscriptionParams)
		})
	})
}

func TestSubscriptionError(t *testing.T) {
	Convey("subscribe to a channel when unconnected", t, func() {
		var fayeClient FayeClient
		var fakeHTTPTransport *FakeHTTPTransport
		var subscriptionParams map[string]interface{}
		var failedResponse Response
		var clientID = "client1"
		var err error

		failedResponse = Response(msgWrapper{&message{ID: "1", Channel: "/meta/subscribe", Successful: false, ClientID: clientID, SupportedConnectionTypes: []string{"long-polling"}}})
		fakeHTTPTransport = &FakeHTTPTransport{usable: true, response: failedResponse}
		registeredTransports = []Transport{fakeHTTPTransport}
		fayeClient = BuildFayeClient().WithTransport(fakeHTTPTransport).Client()
		fayeClient.state = CONNECTED
		subscriptionParams = map[string]interface{}{"channel": "/meta/subscribe", "clientId": clientID, "subscription": "/foo/*", "id": "1"}
		msgChan, err := fayeClient.Subscribe("/foo/*")
		Convey("fails to subscribe", func() {
			So(err, ShouldNotEqual, nil)
			So(msgChan, ShouldBeNil)
		})
		Convey("not add the subscription to the client", func() {
			So(len(fayeClient.subscriptions), ShouldEqual, 0)
		})
		Convey("the client send the subscription to the server", func() {
			So(fakeHTTPTransport.sentParams, ShouldResemble, subscriptionParams)
		})
	})
}

func TestPerformHandshake(t *testing.T) {
	Convey("successful handshake with server", t, func() {
		var fayeClient FayeClient
		var fakeHTTPTransport *FakeHTTPTransport
		var handshakeParams map[string]interface{}
		var response Response
		handshakeParams = map[string]interface{}{"channel": "/meta/handshake",
			"version":                  "1.0",
			"supportedConnectionTypes": []string{"long-polling"}}

		response = Response(msgWrapper{&message{ID: "1", Channel: "/meta/handshake", Successful: true, ClientID: "client4", SupportedConnectionTypes: []string{"long-polling"}}})
		fakeHTTPTransport = &FakeHTTPTransport{usable: true, response: response}
		registeredTransports = []Transport{fakeHTTPTransport}
		fayeClient = BuildFayeClient().Client()
		fayeClient.handshake()
		So(fayeClient.state, ShouldEqual, CONNECTED)
		So(fayeClient.transport, ShouldEqual, fakeHTTPTransport)
		So(fayeClient.clientID, ShouldEqual, "client4")
		So(fakeHTTPTransport.sentParams, ShouldResemble, handshakeParams)
		So(fakeHTTPTransport.url, ShouldEqual, fayeClient.url)
	})

	Convey("unsuccessful handshake with server", t, func() {
		// handshakeParams = map[string]interface{}{
		// 	"channel":                  "/meta/handshake",
		// 	"version":                  "1.0",
		// 	"supportedConnectionTypes": []string{"long-polling"},
		// }

		response := Response(msgWrapper{&message{ID: "1", Channel: "/meta/handshake", Successful: false, ClientID: "client4", SupportedConnectionTypes: []string{"long-polling"}}})
		fakeHTTPTransport := &FakeHTTPTransport{usable: true, response: response, err: errors.New("it didny work")}
		registeredTransports = []Transport{fakeHTTPTransport}
		fayeClient := BuildFayeClient().Client()
		err := fayeClient.handshake()
		So(err, ShouldNotBeNil)
		So(fayeClient.state, ShouldEqual, UNCONNECTED)
		So(fayeClient.schedular.delay(), ShouldEqual, 10*time.Second)
	})

	Convey("handshake with no available transports", t, func() {
		var fayeClient FayeClient
		var fakeHTTPTransport *FakeHTTPTransport
		fakeHTTPTransport = &FakeHTTPTransport{usable: false}
		registeredTransports = []Transport{fakeHTTPTransport}
		fayeClient = BuildFayeClient().Client()
		err := fayeClient.handshake()
		So(err, ShouldNotBeNil)
	})
	Convey("when server does not support available transports", t, func() {
		// var handshakeParams map[string]interface{}
		// handshakeParams = map[string]interface{}{
		// 	"channel":                  "/meta/handshake",
		// 	"version":                  "1.0",
		// 	"supportedConnectionTypes": []string{"long-polling"},
		// }

		response := Response(msgWrapper{&message{ID: "1", Channel: "/meta/handshake", Successful: true, ClientID: "client4", SupportedConnectionTypes: []string{"web-socket"}}})
		fakeHTTPTransport := &FakeHTTPTransport{usable: true, response: response}
		registeredTransports = []Transport{fakeHTTPTransport}
		fayeClient := BuildFayeClient().Client()
		err := fayeClient.handshake()
		So(err, ShouldNotBeNil)
	})
}

func TestHandleResponse(t *testing.T) {
	Convey("it should receive messages to the correct chans", t, func() {
		chan1 := make(chan Message, 5)
		chan2 := make(chan Message, 5)

		subscriptions := []*Subscription{
			{channel: "/foo/bar", msgChan: chan1},
			{channel: "/foo/*", msgChan: chan2},
		}

		params1 := map[string]interface{}{"foo": "bar"}
		params2 := map[string]interface{}{"baz": "qux"}

		messages := []Message{
			Message(msgWrapper{&message{Channel: "/foo/bar", ID: "1", Data: params1}}),
			Message(msgWrapper{&message{Channel: "/foo/quz", ID: "2", Data: params2}}),
			Message(msgWrapper{&message{Channel: "/other/chan", ID: "3", Data: params2}}),
		}

		fayeClient := BuildFayeClient().WithSubscriptions(subscriptions).Client()
		fayeClient.handleMessages(messages)

		//need a very short sleep in here to allow the go routines to complete
		//as all they are doing is assigning a variable 10 milliseconds shoule be more than enough
		Wait(100 * time.Millisecond)

		So(len(chan1), ShouldEqual, 1)
		So(len(chan2), ShouldEqual, 1)
		So((<-chan1).Data(), ShouldContain, params1)
		So((<-chan2).Data(), ShouldContain, params2)
	})
}

func TestPublish(t *testing.T) {
	Convey("publish message to server", t, func() {
		response := Response(msgWrapper{&message{ID: "1", Channel: "/meta/handshake", Successful: true, ClientID: "client4", SupportedConnectionTypes: []string{"long-polling"}}})
		fakeHTTPTransport := &FakeHTTPTransport{usable: true, response: response}
		registeredTransports = []Transport{fakeHTTPTransport}
		fayeClient := BuildFayeClient().WithTransport(fakeHTTPTransport).Connected().Client()
		data := map[string]interface{}{"hello": "world"}
		fayeClient.Publish("/foo", data)
		So(fakeHTTPTransport.sentParams, ShouldResemble, map[string]interface{}{"channel": "/foo", "data": data, "clientId": fayeClient.clientID})
	})
}

func TestExtensions(t *testing.T) {

}
