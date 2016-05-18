package wray

import (
	"bytes"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestDecodeResponse(t *testing.T) {
	var connectJSON = []byte(`[
    {
      "clientId":"fwyr8myfb8b2jwffr66jpsucnztffmq",
      "channel":"/meta/connect",
      "successful":true,
      "advice":{
        "reconnect":"retry",
        "interval":0,
        "timeout":45000
      }
    },
    {
      "channel":"/foo",
      "data":{
        "hello":"from foo"
      },
      "advice":{}
    }
  ]`)

	Convey("the response should be decoded", t, func() {
		dec := json.NewDecoder(bytes.NewBuffer(connectJSON))
		resp, _, err := decodeResponse(dec)

		So(err, ShouldBeNil)
		So(resp.OK(), ShouldBeTrue)
		So(resp.Channel(), ShouldEqual, "/meta/connect")
	})

	Convey("the messages should be decoded", t, func() {
		dec := json.NewDecoder(bytes.NewBuffer(connectJSON))
		_, msgs, err := decodeResponse(dec)

		So(err, ShouldBeNil)
		So(len(msgs), ShouldEqual, 1)
		So(msgs[0].Channel(), ShouldEqual, "/foo")
		So(msgs[0].Data(), ShouldEqual, map[string]interface{}{"hello": "from foo"})
	})

}

func TestMessageError(t *testing.T) {
	Convey("the error should be able to be set on a message", t, func() {
		msg := msgWrapper{&message{}}

		So(msg.HasError(), ShouldBeFalse)

		msg.SetError("test")

		So(msg.HasError(), ShouldBeTrue)
		So(msg.Error(), ShouldEqual, "test")
	})
}
