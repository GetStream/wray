package wray

import (
	"bytes"
	"encoding/json"
	"errors"
	// "fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// HTTPTransport models a faye protocol transport over HTTP long polling
type HTTPTransport struct {
	url string
}

func (t HTTPTransport) isUsable(clientURL string) bool {
	parsedURL, err := url.Parse(clientURL)
	if err != nil {
		return false
	}
	if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
		return true
	}
	return false
}

func (t HTTPTransport) connectionType() string {
	return "long-polling"
}

func (t HTTPTransport) send(msg json.Marshaler) (decoder, error) {
	b, err := json.Marshal(msg)

	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer(b)
	responseData, err := http.Post(t.url, "application/json", buffer)
	if err != nil {
		return nil, err
	}
	if responseData.StatusCode != 200 {
		return nil, errors.New(responseData.Status)
	}
	defer responseData.Body.Close()
	jsonData, err := ioutil.ReadAll(responseData.Body)
	if err != nil {
		return nil, err
	}
	// fmt.Println("BUFFFERERERERER!!! ", string(jsonData))
	return json.NewDecoder(bytes.NewBuffer(jsonData)), nil
}

func (t *HTTPTransport) setURL(url string) {
	t.url = url
}
