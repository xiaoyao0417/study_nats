# NATS - Go Client
A [Go](http://golang.org) client for the [NATS messaging system](https://nats.io).

## 安装

```bash
# Go client
go get github.com/nats-io/nats.go/

# Server
go get github.com/nats-io/nats-server
```

When using or transitioning to Go modules support:

```bash
# Go client latest or explicit version
go get github.com/nats-io/nats.go/@latest
go get github.com/nats-io/nats.go/@v1.25.0

# For latest NATS Server, add /v2 at the end
go get github.com/nats-io/nats-server/v2

# NATS Server v1 is installed otherwise
# go get github.com/nats-io/nats-server
```

## 基本用法

```go
import "github.com/nats-io/nats.go"

// 连接到服务器
nc, _ := nats.Connect(nats.DefaultURL)

// 简单发布
nc.Publish("foo", []byte("Hello World"))

// 简单异步订阅
nc.Subscribe("foo", func(m *nats.Msg) {
    fmt.Printf("Received a message: %s\n", string(m.Data))
})

// 响应请求消息
nc.Subscribe("request", func(m *nats.Msg) {
    m.Respond([]byte("answer is 42"))
})

// 简单同步订阅
sub, err := nc.SubscribeSync("foo")
m, err := sub.NextMsg(timeout)

// 通道订阅
ch := make(chan *nats.Msg, 64)
sub, err := nc.ChanSubscribe("foo", ch)
msg := <- ch

// 取消订阅
sub.Unsubscribe()

// Drain
// 将删除关注，直到所有消息已处理
sub.Drain()

// 请求
msg, err := nc.Request("help", []byte("help me"), 10*time.Millisecond)

// 答复
nc.Subscribe("help", func(m *nats.Msg) {
    nc.Publish(m.Reply, []byte("I can help!"))
})

// Drain connection (Preferred for responders)
// Close() not needed if this is called.
// 进入到Drain状态，完成后将  发布商将被耗尽，无法发布
nc.Drain()

// 关闭连接
nc.Close()
```

## JetStream 基本用法

```go
import "github.com/nats-io/nats.go"

// 连接到NATS
nc, _ := nats.Connect(nats.DefaultURL)

// Create JetStream Context
js, _ := nc.JetStream(nats.PublishAsyncMaxPending(256))

// Simple Stream Publisher
js.Publish("ORDERS.scratch", []byte("hello"))

// Simple Async Stream Publisher
for i := 0; i < 500; i++ {
	js.PublishAsync("ORDERS.scratch", []byte("hello"))
}

select {
case <-js.PublishAsyncComplete():
case <-time.After(5 * time.Second):
	fmt.Println("Did not resolve in time")
}

// Simple Async Ephemeral Consumer
js.Subscribe("ORDERS.*", func(m *nats.Msg) {
	fmt.Printf("Received a JetStream message: %s\n", string(m.Data))
})

// Simple Sync Durable Consumer (optional SubOpts at the end)
sub, err := js.SubscribeSync("ORDERS.*", nats.Durable("MONITOR"), nats.MaxDeliver(3))
m, err := sub.NextMsg(timeout)

// Simple Pull Consumer
sub, err := js.PullSubscribe("ORDERS.*", "MONITOR")
msgs, err := sub.Fetch(10)

// Unsubscribe
sub.Unsubscribe()

// Drain
sub.Drain()
```

## JetStream 基础管理

```go
import "github.com/nats-io/nats.go"

// Connect to NATS
nc, _ := nats.Connect(nats.DefaultURL)

// Create JetStream Context
js, _ := nc.JetStream()

// Create a Stream
js.AddStream(&nats.StreamConfig{
	Name:     "ORDERS",
	Subjects: []string{"ORDERS.*"},
})

// Update a Stream
js.UpdateStream(&nats.StreamConfig{
	Name:     "ORDERS",
	MaxBytes: 8,
})

// Create a Consumer
js.AddConsumer("ORDERS", &nats.ConsumerConfig{
	Durable: "MONITOR",
})

// Delete Consumer
js.DeleteConsumer("ORDERS", "MONITOR")

// Delete Stream
js.DeleteStream("ORDERS")
```

## 编码连接

```go

nc, _ := nats.Connect(nats.DefaultURL)
// json编码
c, _ := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
defer c.Close()

// Simple Publisher
c.Publish("foo", "Hello World")

// Simple Async Subscriber
c.Subscribe("foo", func(s string) {
    fmt.Printf("Received a message: %s\n", s)
})

// EncodedConn can Publish any raw Go type using the registered Encoder
type person struct {
     Name     string
     Address  string
     Age      int
}

// Go type Subscriber
c.Subscribe("hello", func(p *person) {
    fmt.Printf("Received a person: %+v\n", p)
})

me := &person{Name: "derek", Age: 22, Address: "140 New Montgomery Street, San Francisco, CA"}

// Go type Publisher
c.Publish("hello", me)

// Unsubscribe
sub, err := c.Subscribe("foo", nil)
// ...
sub.Unsubscribe()

// Requests
var response string
err = c.Request("help", "help me", &response, 10*time.Millisecond)
if err != nil {
    fmt.Printf("Request failed: %v\n", err)
}

// Replying
c.Subscribe("help", func(subj, reply string, msg string) {
    c.Publish(reply, "I can help!")
})

// Close connection
c.Close();
```

## 身份验证 (Nkeys and 用户凭据)
This requires server with version >= 2.0.0

NATS servers have a new security and authentication mechanism to authenticate with user credentials and Nkeys.
The simplest form is to use the helper method UserCredentials(credsFilepath).
```go
nc, err := nats.Connect(url, nats.UserCredentials("user.creds"))
```

The helper methods creates two callback handlers to present the user JWT and sign the nonce challenge from the server.
The core client library never has direct access to your private key and simply performs the callback for signing the server challenge.
The helper will load and wipe and erase memory it uses for each connect or reconnect.

The helper also can take two entries, one for the JWT and one for the NKey seed file.
```go
nc, err := nats.Connect(url, nats.UserCredentials("user.jwt", "user.nk"))
```

You can also set the callback handlers directly and manage challenge signing directly.
```go
nc, err := nats.Connect(url, nats.UserJWT(jwtCB, sigCB))
```

Bare Nkeys are also supported. The nkey seed should be in a read only file, e.g. seed.txt
```bash
> cat seed.txt
# This is my seed nkey!
SUAGMJH5XLGZKQQWAWKRZJIGMOU4HPFUYLXJMXOO5NLFEO2OOQJ5LPRDPM
```

This is a helper function which will load and decode and do the proper signing for the server nonce.
It will clear memory in between invocations.
You can choose to use the low level option and provide the public key and a signature callback on your own.

```go
opt, err := nats.NkeyOptionFromSeed("seed.txt")
nc, err := nats.Connect(serverUrl, opt)

// Direct
nc, err := nats.Connect(serverUrl, nats.Nkey(pubNkey, sigCB))
```

## TLS

```go
// tls as a scheme will enable secure connections by default. This will also verify the server name.
nc, err := nats.Connect("tls://nats.demo.io:4443")

// If you are using a self-signed certificate, you need to have a tls.Config with RootCAs setup.
// We provide a helper method to make this case easier.
nc, err = nats.Connect("tls://localhost:4443", nats.RootCAs("./configs/certs/ca.pem"))

// If the server requires client certificate, there is an helper function for that too:
cert := nats.ClientCert("./configs/certs/client-cert.pem", "./configs/certs/client-key.pem")
nc, err = nats.Connect("tls://localhost:4443", cert)

// You can also supply a complete tls.Config

certFile := "./configs/certs/client-cert.pem"
keyFile := "./configs/certs/client-key.pem"
cert, err := tls.LoadX509KeyPair(certFile, keyFile)
if err != nil {
    t.Fatalf("error parsing X509 certificate/key pair: %v", err)
}

config := &tls.Config{
    ServerName: 	opts.Host,
    Certificates: 	[]tls.Certificate{cert},
    RootCAs:    	pool,
    MinVersion: 	tls.VersionTLS12,
}

nc, err = nats.Connect("nats://localhost:4443", nats.Secure(config))
if err != nil {
	t.Fatalf("Got an error on Connect with Secure Options: %+v\n", err)
}

```

## 使用Go通道(netchan)

```go
nc, _ := nats.Connect(nats.DefaultURL)

// 创建编码连接
ec, _ := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
defer ec.Close()

type person struct {
     Name     string
     Address  string
     Age      int
}

recvCh := make(chan *person)
// 绑定接收通道
ec.BindRecvChan("hello", recvCh)

sendCh := make(chan *person)
// 绑定发送通道
ec.BindSendChan("hello", sendCh)

me := &person{Name: "derek", Age: 22, Address: "140 New Montgomery Street"}

// 通过Go通道发送
sendCh <- me

// 通过Go通道接收
who := <- recvCh
```

## 通配符订阅

```go

// “*”匹配主题任何级别的任何标记
nc.Subscribe("foo.*.baz", func(m *Msg) {
    fmt.Printf("Msg received on [%s] : %s\n", m.Subject, string(m.Data));
})

nc.Subscribe("foo.bar.*", func(m *Msg) {
    fmt.Printf("Msg received on [%s] : %s\n", m.Subject, string(m.Data));
})

//“>”匹配主题尾部的任何长度，并且只能是最后一个标记
//例如“foo.>'将匹配“foo.bar”、“foo.bar.baz”、“foo.foo.bar.bax.22'”
nc.Subscribe("foo.>", func(m *Msg) {
    fmt.Printf("Msg received on [%s] : %s\n", m.Subject, string(m.Data));
})

// 匹配以上所有内容
nc.Publish("foo.bar.baz", []byte("Hello World"))

```

## 队列组

```go
// All subscriptions with the same queue name will form a queue group.
// Each message will be delivered to only one subscriber per queue group,
// using queuing semantics. You can have as many queue groups as you wish.
// Normal subscribers will continue to work as expected.

//具有相同队列名称的所有订阅都将组成一个队列组。
//每个消息将被传递给每个队列组仅一个订户，
//使用排队语义。您可以拥有任意数量的队列组。
//普通订户将继续按预期工作。

nc.QueueSubscribe("foo", "job_workers", func(_ *Msg) {
  received += 1;
})
```

## 高级用法

```go

// 通常，库会在尝试连接没有运行的服务器时返回一个错误。RetryOnFailedConnect 选项将设置如果无法立即连接，则连接处于重新连接状态。

nc, err := nats.Connect(nats.DefaultURL,
    nats.RetryOnFailedConnect(true),
    nats.MaxReconnects(10),
    nats.ReconnectWait(time.Second),
    nats.ReconnectHandler(func(_ *nats.Conn) {
        // 请注意，这将用于第一个异步连接。.
    }))
if err != nil {
    //即使无法连接，也不应返回错误，但是您仍然需要检查是否有一些配置错误。
}

// 与服务器的Flush连接，当所有消息都处理时返回。
nc.Flush()
fmt.Println("All clear!")

// FlushTimeOut也指定超时值。
err := nc.FlushTimeout(1*time.Second)
if err != nil {
    fmt.Println("All clear!")
} else {
    fmt.Println("Flushed timed out!")
}

// Auto-unsubscribe after MAX_WANTED messages received

// 接收到max_want的消息后自动Auto-unsubscribe
const MAX_WANTED = 10
sub, err := nc.Subscribe("foo")
sub.AutoUnsubscribe(MAX_WANTED)

// 多个连接
nc1 := nats.Connect("nats://host1:4222")
nc2 := nats.Connect("nats://host2:4222")

nc1.Subscribe("foo", func(m *Msg) {
    fmt.Printf("Received a message: %s\n", string(m.Data))
})

nc2.Publish("foo", []byte("Hello World!"));

```

## 集群使用

```go

var servers = "nats://localhost:1222, nats://localhost:1223, nats://localhost:1224"

nc, err := nats.Connect(servers)

//可选设置ReconnectWait和MaxReconnect尝试。 
//此示例意味着每个后端总共10秒。
nc, err = nats.Connect(servers, nats.MaxReconnects(5), nats.ReconnectWait(2 * time.Second))

// You can also add some jitter for the reconnection.
// This call will add up to 500 milliseconds for non TLS connections and 2 seconds for TLS connections.
// If not specified, the library defaults to 100 milliseconds and 1 second, respectively.
//您还可以为重新连接添加一些抖动。 
 //此呼叫的非TLS连接最多可高达500毫秒，而TLS连接的2秒。 
 //如果未指定，库默认为100毫秒和1秒。
nc, err = nats.Connect(servers, nats.ReconnectJitter(500*time.Millisecond, 2*time.Second))

// You can also specify a custom reconnect delay handler. If set, the library will invoke it when it has tried
// all URLs in its list. The value returned will be used as the total sleep time, so add your own jitter.
// The library will pass the number of times it went through the whole list.
//您还可以指定自定义重新连接延迟处理程序。 如果设置，库在尝试时会调用它 
 //列表中的所有URL。 返回的值将用作总睡眠时间，因此请添加您自己的抖动。 
 //图书馆将通过整个列表的次数。
nc, err = nats.Connect(servers, nats.CustomReconnectDelay(func(attempts int) time.Duration {
    return someBackoffFunction(attempts)
}))

// Optionally disable randomization of the server pool
nc, err = nats.Connect(servers, nats.DontRandomize())

// Setup callbacks to be notified on disconnects, reconnects and connection closed.
nc, err = nats.Connect(servers,
	nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		fmt.Printf("Got disconnected! Reason: %q\n", err)
	}),
	nats.ReconnectHandler(func(nc *nats.Conn) {
		fmt.Printf("Got reconnected to %v!\n", nc.ConnectedUrl())
	}),
	nats.ClosedHandler(func(nc *nats.Conn) {
		fmt.Printf("Connection closed. Reason: %q\n", nc.LastError())
	})
)

// When connecting to a mesh of servers with auto-discovery capabilities,
// you may need to provide a username/password or token in order to connect
// to any server in that mesh when authentication is required.
// Instead of providing the credentials in the initial URL, you will use
// new option setters:
nc, err = nats.Connect("nats://localhost:4222", nats.UserInfo("foo", "bar"))

// For token based authentication:
nc, err = nats.Connect("nats://localhost:4222", nats.Token("S3cretT0ken"))

// You can even pass the two at the same time in case one of the server
// in the mesh requires token instead of user name and password.
nc, err = nats.Connect("nats://localhost:4222",
    nats.UserInfo("foo", "bar"),
    nats.Token("S3cretT0ken"))

// Note that if credentials are specified in the initial URLs, they take
// precedence on the credentials specified through the options.
// For instance, in the connect call below, the client library will use
// the user "my" and password "pwd" to connect to localhost:4222, however,
// it will use username "foo" and password "bar" when (re)connecting to
// a different server URL that it got as part of the auto-discovery.
nc, err = nats.Connect("nats://my:pwd@localhost:4222", nats.UserInfo("foo", "bar"))

```

## Context 支持

```go

ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

nc, err := nats.Connect(nats.DefaultURL)

// Request with context
msg, err := nc.RequestWithContext(ctx, "foo", []byte("bar"))

// Synchronous subscriber with context
sub, err := nc.SubscribeSync("foo")
msg, err := sub.NextMsgWithContext(ctx)

// Encoded Request with context
c, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
type request struct {
	Message string `json:"message"`
}
type response struct {
	Code int `json:"code"`
}
req := &request{Message: "Hello"}
resp := &response{}
err := c.RequestWithContext(ctx, "foo", req, resp)
```
