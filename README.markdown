# Wray, faye client for Go

Wray is a client for the [Faye](http://faye.jcoglan.com) publish-subscribe messaging service by [James Coglan](https://twitter.com/jcoglan).  Originally
created by [pythonandchips](https://github.com/pythonandchips), this fork is now maintained by Autogrow Systems Limited to serve internal application
development.

## Current status

**Version 2 Beta**.  Should work but might be broken in some places.  Only long polling is supported at this time.

This is compatible with the gopkg.in method of importing: `gopkg.in/autogrowsystems/wray.v2`

* **v2** - breaking changes, public API changed
* **v1** - breaking changes, public API changed
* **v0** - initial fork, plus some mods

## Getting Started

Wray is only a client for Faye. You will need to setup a server using Ruby or Node.js first. Instructions that can be found on the [Faye project pages](http://faye.jcoglan.com/).  Simple examples are available in the examples folder.

This is the standard stuff you will need to do to use wray:

```go
import "gopkg.in/autogrowsystems/wray.v2"

// register the types of transport you want available. Only long-polling is currently supported
wray.RegisterTransports([]wray.Transport{ &wray.HTTPTransport{} }) // probably best put this in your init() func

client := wray.NewFayeClient("http://localhost:8000/faye")
go client.Listen() // start the listen loop

```

### Subscribing to channels

This is some sample code for subscribing to a channel.  Wildcard channels ar permissible too (eg: `/foo/*`).

The `WaitSubscribe()` method allows you to ensure that the subscription worked by blocking until .

This method allows you get your Faye messages through a go channel, which allows you
to select on multiple subscriptions and other sources:

```go
msgChan, err := client.Subscribe("/foo")
if err != nil {
  panic(err)
}

// similar to above
for {
  msg <- msgChan
  fmt.Println("-------------------------------------------")
  fmt.Println(msg.Data())
}
```

Or a more complex setup:

```go
fooChan := client.WaitSubscribe("/foo")
barChan := client.WaitSubscribe("/bar")
bazChan := client.WaitSubscribe("/baz")

for {
  select {
  case msg := <-fooChan
    // do something with the message
  case msg := <-barChan
    // do something with the message
  case msg := <-bazChan
    // do something with the message
  case <- ticker.C
    // do something with the ticker channel
  }
}
```

Finally you can give your own channel to tell the subscribe operation to fork off:

```go
go client.WaitSubscribe("/foo", mything.fooChan)
go client.WaitSubscribe("/bar", mything.barChan)
go client.WaitSubscribe("/baz", mything.bazChan)
go client.WaitSubscribe("/baz/stuff", mything.bazChan)

for {
  select {
  case msg := <-mything.fooChan
    // do something with the message
  case msg := <-mything.barChan
    // do something with the message
  case msg := <-mything.bazChan
    // do something with the message
  case <- ticker.C
    // do something with the ticker channel
  }
}
```

### Publishing to channels

Publishing is pretty straight forward:

```go
params := map[string]interface{}{"hello": "faye"}

//send message to server
if err := client.Publish("/foo", params); err != nil {
  panic(err)
}
```

### Extensions

You can make extensions that satisfy the `faye.Extension` interface:

```go
type Extension interface {
  In(Message)
  Out(Message)
}
```

Then you can write your extension like so:

```go
type authExtension struct {
  token string
}

// In does nothing in this extension, but is needed to satisy the interface
func (e *authExtension) In(msg wray.Message) {}

// Out adds the authentication token to the messages ext field
func (e *authExtension) Out(msg wray.Message) {
  ext := msg.Ext()
  ext["token"] = e.token
}

client.AddExtension(authExtension{"Y29tZSBhdCBtZSBicm8K"})

// this message will now reach the server with the authentication token attached
client.Publish("/test", map[string]interface{}{"hello": "world"})
```

The client even implements the extension interface so you can feed it back to itself
to spy on the messages going in and out (requires logger to be at DEBUG level):

```go
client.AddExtension(client)
// 2016/05/19 07:46:20 [DEBUG] : Transporting message
// 2016/05/19 07:46:20 [DEBUG] : Running extension *wray.FayeClient in
// 2016/05/19 07:46:20 [DEBUG] : [SPY-IN]  {"channel":"/foo","successful":true,"advice":{}}
// 2016/05/19 07:46:22 [DEBUG] : Running extension *wray.FayeClient out
// 2016/05/19 07:46:22 [DEBUG] : [SPY-OUT] {"channel":"/foo","clientId":"dllmvy7z4j7l8hdafkknu0q1yasnv9d","data":{"hello":"from golang"},"advice":{}}
// 2016/05/19 07:46:22 [DEBUG] : Transporting message
// 2016/05/19 07:46:22 [DEBUG] : Running extension *wray.FayeClient in
// 2016/05/19 07:46:22 [DEBUG] : [SPY-IN]  {"channel":"/foo","successful":true,"advice":{}}
```

### Logging

The client uses the default go logger and outputs to STDOUT but if you want to give
your own you can use the `SetLogger()` method.  Just make sure it satisfies the
`faye.Logger` interface:

```go
type Logger interface {
	Infof(f string, a ...interface{})
	Errorf(f string, a ...interface{})
	Debugf(f string, a ...interface{})
	Warnf(f string, a ...interface{})
}
```

## Future Work

There is still a lot to do to bring Wray in line with Faye functionality. This is a less than exhaustive list of work to be completed:-

- [x] fix the resubscribe after rehandshake logic to work with channels instead of callbacks
- [ ] web socket support
- [ ] eventsource support
- [x] logging
- [x] don't panic!
- [x] don't panic when the server goes away
- [ ] fix broken tests
- [ ] stop having to register transports
- [x] subscribe by giving a channel name and a go channel
- [x] extensions support
- [ ] correctly handle disconnect and server down
- [ ] automated integrations test to ensure Wray continues to work with Faye
- [ ] split connection out into it's own object
- [ ] timeout connect requests using advice from the server
- [ ] return an interface instead of the client itself
- [ ] use NewClient instead of NewFayeClient

## Contributing

* Check out the latest master to make sure the feature hasn't been implemented or the bug hasn't been fixed yet
* Check out the issue tracker to make sure someone already hasn't requested it and/or contributed it
* Fork the project
* Start a feature/bugfix branch
* Commit and push until you are happy with your contribution
* Make sure to add tests for it. This is important so I don't break it in a future version unintentionally.

## Copyright

Copyright (c) 2016 Colin Gemmell, Robert McLeod, Autogrow Systems Limited

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
