package wray

import (
	"bytes"
	"encoding/json"
)

type FakeHTTPTransport struct {
	usable     bool
	timesSent  int
	sentParams map[string]interface{}
	response   Response
	url        string
	err        error
}

func (fake FakeHTTPTransport) isUsable(endpoint string) bool {
	return fake.usable
}

func (fake FakeHTTPTransport) connectionType() string {
	return "long-polling"
}

func (fake *FakeHTTPTransport) send(json.Marshaler) (decoder, error) {
	fake.timesSent++
	return json.NewDecoder(bytes.NewBuffer([]byte{})), fake.err
}

func (fake *FakeHTTPTransport) setURL(url string) {
	fake.url = url
}
