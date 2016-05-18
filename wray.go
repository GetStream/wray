package wray

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	UNCONNECTED  = 1
	CONNECTING   = 2
	CONNECTED    = 3
	DISCONNECTED = 4

	HANDSHAKE = "handshake"
	RETRY     = "retry"
	NONE      = "none"

	CONNECTION_TIMEOUT = 60.0
	DEFAULT_RETRY      = 5.0
	MAX_REQUEST_SIZE   = 2048
)

var (
	MANDATORY_CONNECTION_TYPES = []string{"long-polling"}
	registeredTransports       = []Transport{}
)

// Extension models a faye extension
type Extension interface {
	In(Message)
	Out(Message)
}

// Subscription models a subscription, containing the channel it is subscribed
// to and the chan object used to push messages through
type Subscription struct {
	channel string
	msgChan chan Message
}

// Logger is the interface that faye uses for it's logger
type Logger interface {
	Infof(f string, a ...interface{})
	Errorf(f string, a ...interface{})
	Debugf(f string, a ...interface{})
	Warnf(f string, a ...interface{})
}

type fayeLogger struct{}

func (l fayeLogger) Infof(f string, a ...interface{}) {
	log.Printf("[INFO]  : "+f, a...)
}
func (l fayeLogger) Errorf(f string, a ...interface{}) {
	log.Printf("[ERROR] : "+f, a...)
}
func (l fayeLogger) Debugf(f string, a ...interface{}) {
	log.Printf("[DEBUG] : "+f, a...)
}
func (l fayeLogger) Warnf(f string, a ...interface{}) {
	log.Printf("[WARN]  : "+f, a...)
}

// FayeClient models a faye client
type FayeClient struct {
	state          int
	url            string
	subscriptions  []*Subscription
	transport      Transport
	log            Logger
	clientID       string
	schedular      schedular
	nextRetry      int64
	nextHandshake  int64
	mutex          *sync.RWMutex // protects instance vars across goroutines
	connectMutex   *sync.RWMutex // ensures a single connection to the server as per the protocol
	handshakeMutex *sync.RWMutex // ensures only a single handshake at a time
	extns          []Extension
}

// NewFayeClient returns a new client for interfacing to a faye server
func NewFayeClient(url string) *FayeClient {
	return &FayeClient{
		url:            url,
		state:          UNCONNECTED,
		schedular:      channelSchedular{},
		mutex:          &sync.RWMutex{},
		connectMutex:   &sync.RWMutex{},
		handshakeMutex: &sync.RWMutex{},
		log:            fayeLogger{},
	}
}

// Out spies on outgoing messages and prints them to the log, when the client
// is added to itself as an extension
func (faye *FayeClient) Out(msg Message) {
	switch v := msg.(type) {
	case msgWrapper:
		b, _ := v.MarshalJSON()
		faye.log.Debugf("[SPY-OUT] %s\n", string(b))
	}
}

// In spies on outgoing messages and prints them to the log, when the client
// is added to itself as an extension
func (faye *FayeClient) In(msg Message) {
	switch v := msg.(type) {
	case msgWrapper:
		b, _ := v.MarshalJSON()
		faye.log.Debugf("[SPY-IN]  %s\n", string(b))
	}
}

// SetLogger attaches a Logger to the faye client, and replaces the default
// logger which just puts to stdout
func (faye *FayeClient) SetLogger(log Logger) {
	faye.log = log
}

func (faye *FayeClient) connectIfNotConnected() error {
	faye.whileConnectingBlockUntilConnected()
	if faye.state == UNCONNECTED {
		if err := faye.handshake(); err != nil {
			return err
		}
	}

	return nil
}

func (faye *FayeClient) whileConnectingBlockUntilConnected() {
	if faye.state == CONNECTING {
		for {
			if faye.state == CONNECTED {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func (faye *FayeClient) handshake() error {

	faye.handshakeMutex.Lock()
	defer faye.handshakeMutex.Unlock()

	// uh oh spaghettios!
	if faye.state == DISCONNECTED {
		return fmt.Errorf("GTFO: Server told us not to reconnect :(")
	}

	// check if we need to wait before handshaking again
	if faye.nextHandshake > time.Now().Unix() {
		sleepFor := time.Now().Unix() - faye.nextHandshake

		// wait for the duration the server told us
		if sleepFor > 0 {
			faye.log.Debugf("Waiting for", sleepFor, "seconds before next handshake")
			time.Sleep(time.Duration(sleepFor) * time.Second)
		}
	}

	faye.log.Debugf("Handshaking....")

	t, err := selectTransport(faye, MANDATORY_CONNECTION_TYPES, []string{})
	if err != nil {
		return fmt.Errorf("No usable transports available")
	}

	faye.mutex.Lock()
	faye.transport = t
	faye.transport.setURL(faye.url)
	faye.state = CONNECTING
	faye.mutex.Unlock()

	var response Response

	for {
		msg := faye.newMessage("/meta/handshake")
		msg.Version = "1.0"
		msg.SupportedConnectionTypes = []string{"long-polling"}
		response, _, err = faye.send(msg)

		if err != nil {
			faye.mutex.Lock()
			faye.state = UNCONNECTED
			faye.mutex.Unlock()

			faye.log.Warnf("Handshake failed. Retry in 10 seconds")
			time.Sleep(10 * time.Second)
			continue
		}

		break
	}

	faye.mutex.Lock()
	oldClientID := faye.clientID
	faye.clientID = response.ClientID()
	faye.state = CONNECTED
	faye.transport, err = selectTransport(faye, response.SupportedConnectionTypes(), []string{})
	faye.mutex.Unlock()

	if err != nil {
		return fmt.Errorf("Server does not support any available transports. Supported transports: " + strings.Join(response.SupportedConnectionTypes(), ","))
	}

	if oldClientID != faye.clientID && len(faye.subscriptions) > 0 {
		faye.log.Warnf("Client ID changed (%s => %s), %d invalid subscriptions\n", oldClientID, faye.clientID, len(faye.subscriptions))
		faye.resubscribeAll()
	}

	return nil
}

// change the state in a thread safe manner
func (faye *FayeClient) changeState(state int) {
	faye.mutex.Lock()
	defer faye.mutex.Unlock()
	faye.state = state
}

// TODO: check the bayeux spec to see if the retry period counts for all requests
// func (faye *FayeClient) makeRequest(data map[string]interface{}) (Response, error) {
// 	faye.connectMutex.Lock()
// 	defer faye.connectMutex.Unlock()
//
// 	// wait to retry if we were told to
// 	if faye.nextRetry > time.Now().Unix() {
// 		sleepFor := faye.nextRetry - time.Now().Unix()
// 		if sleepFor > 0 {
// 			// fmt.Println("Waiting for", sleepFor, "seconds before connecting")
// 			time.Sleep(time.Duration(sleepFor) * time.Second)
// 		}
// 	}
//
// 	return faye.transport.send(subscriptionParams)
// }

// resubscribe all of the subscriptions
func (faye *FayeClient) resubscribeAll() {

	faye.mutex.Lock()
	subs := faye.subscriptions
	faye.subscriptions = []*Subscription{}
	faye.mutex.Unlock()

	faye.log.Debugf("Attempting to resubscribe %d subscriptions\n", len(subs))
	for _, sub := range subs {

		// fork off all the resubscribe requests
		go func(sub *Subscription) {
			for {
				err := faye.requestSubscription(sub.channel)

				// if it worked add it back to the list
				if err == nil {
					faye.mutex.Lock()
					defer faye.mutex.Unlock()
					faye.subscriptions = append(faye.subscriptions, sub)

					faye.log.Debugf("Resubscribed to %s", sub.channel)
					return
				}

				time.Sleep(500 * time.Millisecond)
			}
		}(sub)

	}
}

// requests a subscription from the server and returns error if the request failed
func (faye *FayeClient) requestSubscription(channel string) error {
	if err := faye.connectIfNotConnected(); err != nil {
		return err
	}

	msg := faye.newMessage("/meta/subscribe")
	msg.Subscription = channel

	// TODO: check if the protocol allows a subscribe during an active connect request
	response, _, err := faye.send(msg)
	if err != nil {
		return err
	}

	go faye.handleAdvice(response.Advice())

	if !response.OK() {
		// TODO: put more information in the error message about why it failed
		errmsg := "Response was unsuccessful: "
		if err != nil {
			errmsg += err.Error()
		}

		if response.HasError() {
			errmsg += " / " + response.Error()
		}
		reserr := errors.New(errmsg)
		return reserr
	}

	return nil
}

func (faye *FayeClient) newMessage(channel string) *message {
	return &message{
		ClientID: faye.clientID,
		Channel:  channel,
	}
}

// handles a response from the server
func (faye *FayeClient) handleMessages(msgs []Message) {
	for _, message := range msgs {
		faye.runExtensions("in", message)
		for _, subscription := range faye.subscriptions {
			matched, _ := filepath.Match(subscription.channel, message.Channel())
			if matched {
				go func() { subscription.msgChan <- message }()
			}
		}
	}
}

// handles advice from the server
func (faye *FayeClient) handleAdvice(advice Advice) {
	faye.mutex.Lock()
	defer faye.mutex.Unlock()

	if advice.Reconnect() != "" {
		interval := advice.Interval()

		switch advice.Reconnect() {
		case RETRY:
			if interval > 0 {
				faye.nextHandshake = int64(time.Duration(time.Now().Unix()) + (time.Duration(interval) * time.Millisecond))
			}
		case HANDSHAKE:
			faye.state = UNCONNECTED // force a handshake on the next request
			if interval > 0 {
				faye.nextHandshake = int64(time.Duration(time.Now().Unix()) + (time.Duration(interval) * time.Millisecond))
			}
		case NONE:
			faye.state = DISCONNECTED
			faye.log.Errorf("GTFO: Server advised not to reconnect :(")
		}
	}
}

// connects to the server and waits for a response.  Will block if it is waiting
// for the nextRetry time as advised by the server.  This locks the connectMutex
// so other connections can't go through until the
func (faye *FayeClient) connect() {
	faye.connectMutex.Lock()
	defer faye.connectMutex.Unlock()

	msg := faye.newMessage("/meta/connect")
	msg.ConnectionType = faye.transport.connectionType()

	response, messages, err := faye.send(msg)
	if err != nil {
		faye.log.Errorf("Error while sending connect request: %s", err)
		return
	}

	go faye.handleAdvice(response.Advice())

	if response.OK() {
		go faye.handleMessages(messages)
	} else {
		faye.log.Errorf("Error in response to connect request: %s", response.Error())
	}

}

// wraps the call to transport.send()
func (faye *FayeClient) send(msg *message) (Response, []Message, error) {
	if msg.ClientID == "" && msg.Channel != "/meta/handshake" && faye.clientID != "" {
		msg.ClientID = faye.clientID
	}

	message := Message(msgWrapper{msg})
	faye.runExtensions("out", message)

	if message.HasError() {
		return nil, []Message{}, message // Message has Error() so can be returned as an error
	}

	dec, err := faye.transport.send(message)
	if err != nil {
		err = fmt.Errorf("Error transporting message: %s", err)
		faye.log.Errorf("%s", err)
		return nil, []Message{}, err
	}

	r, m, err := decodeResponse(dec)
	if err != nil {
		faye.log.Errorf("Failed to decode response: %s", err)
	}

	if r != nil {
		faye.runExtensions("in", r.(Message))
	}

	return r, m, err
}

// AddExtension adds an extension to the Faye Client
func (faye *FayeClient) AddExtension(extn Extension) {
	faye.extns = append(faye.extns, extn)
}

func (faye *FayeClient) runExtensions(direction string, msg Message) {
	for _, extn := range faye.extns {
		faye.log.Debugf("Running extension %T %s", extn, direction)
		switch direction {
		case "out":
			extn.Out(msg)
		case "in":
			extn.In(msg)
		}
	}
}

// Subscribe returns a channel to receive messages through
func (faye *FayeClient) Subscribe(channel string) (chan Message, error) {
	if err := faye.requestSubscription(channel); err != nil {
		return nil, err
	}

	msgChan := make(chan Message)
	subscription := &Subscription{channel: channel, msgChan: msgChan}

	faye.mutex.Lock()
	defer faye.mutex.Unlock()
	faye.subscriptions = append(faye.subscriptions, subscription)

	return msgChan, nil
}

// WaitSubscribe will send a subscribe request and block until the connection was successful
func (faye *FayeClient) WaitSubscribe(channel string, optionalMsgChan ...chan Message) chan Message {

	for {
		if err := faye.requestSubscription(channel); err != nil {
			faye.log.Errorf("[WaitSubscribe]: %s", err)
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	msgChan := make(chan Message)
	if len(optionalMsgChan) > 0 {
		msgChan = optionalMsgChan[0]
	}

	subscription := &Subscription{channel: channel, msgChan: msgChan}

	faye.mutex.Lock()
	defer faye.mutex.Unlock()
	faye.subscriptions = append(faye.subscriptions, subscription)

	return msgChan
}

// Publish a message to the given channel
func (faye *FayeClient) Publish(channel string, data map[string]interface{}) error {
	if err := faye.connectIfNotConnected(); err != nil {
		return err
	}

	msg := faye.newMessage(channel)
	msg.Data = data
	response, _, err := faye.send(msg)
	if err != nil {
		return err
	}

	go faye.handleAdvice(response.Advice())

	if !response.OK() {
		return fmt.Errorf("Response was not successful")
	}

	return nil
}

// Listen starts listening for subscription requests from the server.  It is
// blocking but can safely run in it's own goroutine.
func (faye *FayeClient) Listen() {
	for {
		if err := faye.connectIfNotConnected(); err != nil {
			faye.log.Errorf("Handshake failed: %s", err)
			time.Sleep(5 * time.Second)
		}

		for {
			if faye.state != CONNECTED {
				break
			}

			// wait to retry if we were told to
			if faye.nextRetry > time.Now().Unix() {
				sleepFor := faye.nextRetry - time.Now().Unix()
				if sleepFor > 0 {
					// fmt.Println("Waiting for", sleepFor, "seconds before connecting")
					time.Sleep(time.Duration(sleepFor) * time.Second)
				}
			}

			faye.connect()
		}
	}
}

// RegisterTransports allows for the dynamic loading of different transports
// and the most suitable one will be selected
func RegisterTransports(transports []Transport) {
	registeredTransports = transports
}
