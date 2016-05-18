package wray

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func params2msg(params map[string]interface{}) Message {
	b, err := json.Marshal(params)
	if err != nil {
		panic(err)
	}
	msg := message{}
	json.Unmarshal(b, &msg)
	return msgWrapper{&msg}
}

func TestIsUsable(t *testing.T) {
	Convey("usable if the url is http", t, func() {
		var url string
		var httpTransport HTTPTransport
		var isUsable = true
		url = "http://localhost"
		httpTransport = HTTPTransport{}
		isUsable = httpTransport.isUsable(url)
		So(isUsable, ShouldEqual, true)
	})
	Convey("usable if the url is https", t, func() {
		var url string
		var httpTransport HTTPTransport
		var isUsable = true
		url = "https://localhost"
		httpTransport = HTTPTransport{}
		isUsable = httpTransport.isUsable(url)
		So(isUsable, ShouldEqual, true)
	})
	Convey("usable if the url is ws", t, func() {
		var url string
		var httpTransport HTTPTransport
		var isUsable = true
		url = "ws://localhost"
		httpTransport = HTTPTransport{}
		isUsable = httpTransport.isUsable(url)
		So(isUsable, ShouldEqual, false)
	})
	Convey("usable if the url is not blank", t, func() {
		var url string
		var httpTransport HTTPTransport
		var isUsable = true
		url = ""
		httpTransport = HTTPTransport{}
		isUsable = httpTransport.isUsable(url)
		So(isUsable, ShouldEqual, false)
	})
}

func TestConnectionType(t *testing.T) {
	Convey("connection type is long-polling", t, func() {
		var httpTransport HTTPTransport
		httpTransport = HTTPTransport{}
		So(httpTransport.connectionType(), ShouldEqual, "long-polling")
	})
}
