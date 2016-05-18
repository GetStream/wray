package wray

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestSelectTransport(t *testing.T) {
	Convey("when all transports are usable", t, func() {
		var transportTypes []string
		var transport Transport
		var fakeHTTPTransport *FakeHTTPTransport
		var fayeClient *FayeClient

		transportTypes = []string{"long-polling"}
		fakeHTTPTransport = &FakeHTTPTransport{usable: true}
		fayeClient = &FayeClient{url: "http://localhost"}
		registeredTransports = []Transport{fakeHTTPTransport}
		transport, _ = selectTransport(fayeClient, transportTypes, []string{})
		So(transport, ShouldEqual, fakeHTTPTransport)
	})

	Convey("when no transports are usable", t, func() {
		var transportTypes []string
		var fakeHTTPTransport *FakeHTTPTransport
		var fayeClient *FayeClient
		var err error

		transportTypes = []string{"long-polling"}
		fakeHTTPTransport = &FakeHTTPTransport{usable: false}
		fayeClient = &FayeClient{url: "http://localhost"}
		registeredTransports = []Transport{fakeHTTPTransport}
		_, err = selectTransport(fayeClient, transportTypes, []string{})
		So(err, ShouldNotBeNil)
	})
}
