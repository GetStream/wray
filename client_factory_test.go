package wray

import (
	"sync"
	"time"
)

type FakeSchedular struct {
	requestedDelay time.Duration
}

func (fs *FakeSchedular) wait(delay time.Duration, callback func()) {
	fs.requestedDelay = delay
}

func (fs *FakeSchedular) delay() time.Duration {
	return fs.requestedDelay
}

type FayeClientBuilder struct {
	client FayeClient
}

func BuildFayeClient() FayeClientBuilder {
	schedular := &FakeSchedular{}
	client := FayeClient{state: UNCONNECTED, url: "https://localhost", clientID: "", schedular: schedular, mutex: &sync.RWMutex{}, connectMutex: &sync.RWMutex{}}
	return FayeClientBuilder{client}
}

func (factory FayeClientBuilder) Client() FayeClient {
	return factory.client
}

func (factory FayeClientBuilder) Connected() FayeClientBuilder {
	factory.client.state = CONNECTED
	return factory
}

func (factory FayeClientBuilder) WithTransport(transport Transport) FayeClientBuilder {
	factory.client.transport = transport
	return factory
}

func (factory FayeClientBuilder) WithSubscriptions(subscriptions []*Subscription) FayeClientBuilder {
	factory.client.subscriptions = subscriptions
	return factory
}
