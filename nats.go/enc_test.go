// Copyright 2012-2022 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nats_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/encoders/protobuf"
	"github.com/nats-io/nats.go/encoders/protobuf/testdata"
)

// Since we import above nats packages, we need to have a different
// const name than TEST_PORT that we used on the other packages.
const ENC_TEST_PORT = 8268

var options = Options{
	Url:            fmt.Sprintf("nats://127.0.0.1:%d", ENC_TEST_PORT),
	AllowReconnect: true,
	MaxReconnect:   10,
	ReconnectWait:  100 * time.Millisecond,
	Timeout:        DefaultTimeout,
}

////////////////////////////////////////////////////////////////////////////////
// 编码连接测试
////////////////////////////////////////////////////////////////////////////////

// 使用json编解码
func TestPublishErrorAfterSubscribeDecodeError(t *testing.T) {
	ts := RunServerOnPort(ENC_TEST_PORT)
	defer ts.Shutdown()

	opts := options
	nc, _ := opts.Connect()
	defer nc.Close()

	// 覆盖默认handler进行测试。
	nc.SetErrorHandler(func(_ *Conn, _ *Subscription, _ error) {})

	c, _ := NewEncodedConn(nc, JSON_ENCODER)

	// 测试消息类型
	type Message struct {
		Message string
	}
	const testSubj = "test"

	c.Subscribe(testSubj, func(msg *Message) {})

	// 发布无效json以捕获订阅回调中的解码错误
	c.Publish(testSubj, `foo`)
	c.Flush()

	// 下一个发布将成功
	if err := c.Publish(testSubj, Message{"2"}); err != nil {
		t.Error("Fail to send correct json message after decode error in subscription")
	}
}

// 使用protobuf编解码
func TestPublishErrorAfterInvalidPublishMessage(t *testing.T) {
	ts := RunServerOnPort(ENC_TEST_PORT)
	defer ts.Shutdown()

	opts := options
	nc, _ := opts.Connect()
	defer nc.Close()

	c, _ := NewEncodedConn(nc, protobuf.PROTOBUF_ENCODER)
	const testSubj = "test"

	c.Publish(testSubj, &testdata.Person{Name: "Anatolii"})

	// Publish invalid protobuf message to catch decode error
	c.Publish(testSubj, "foo")

	// Next publish with valid protobuf message should be successful
	if err := c.Publish(testSubj, &testdata.Person{Name: "Anatolii"}); err != nil {
		t.Error("Fail to send correct protobuf message after invalid message publishing", err)
	}
}

func TestVariousFailureConditions(t *testing.T) {
	ts := RunServerOnPort(ENC_TEST_PORT)
	defer ts.Shutdown()

	dch := make(chan bool)

	opts := options
	opts.AsyncErrorCB = func(_ *Conn, _ *Subscription, e error) {
		dch <- true
	}

	nc, _ := opts.Connect()
	nc.Close()

	// 错误: nats连接为空
	if _, err := NewEncodedConn(nil, protobuf.PROTOBUF_ENCODER); err == nil {
		t.Fatal("Expected an error")
	}

	// 错误: nats连接已经关闭
	if _, err := NewEncodedConn(nc, protobuf.PROTOBUF_ENCODER); err == nil || err != ErrConnectionClosed {
		t.Fatalf("Wrong error: %v instead of %v", err, ErrConnectionClosed)
	}

	nc, _ = opts.Connect()
	defer nc.Close()

	// 错误: Encoder 注册失败
	if _, err := NewEncodedConn(nc, "foo"); err == nil {
		t.Fatal("Expected an error")
	}

	c, err := NewEncodedConn(nc, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		t.Fatalf("Unable to create encoded connection: %v", err)
	}
	defer c.Close()

	if _, err := c.Subscribe("bar", func(subj, obj string) {}); err != nil {
		t.Fatalf("Unable to create subscription: %v", err)
	}

	if err := c.Publish("bar", &testdata.Person{Name: "Ivan"}); err != nil {
		t.Fatalf("Unable to publish: %v", err)
	}

	if err := Wait(dch); err != nil {
		t.Fatal("Did not get the async error callback")
	}

	// 错误: 通过编码时，无效的protobuf消息
	if err := c.PublishRequest("foo", "bar", "foo"); err == nil {
		t.Fatal("Expected an error")
	}

	// 错误: 为nill，通过编码时，无效的protobuf消息
	if err := c.Request("foo", "foo", nil, 2*time.Second); err == nil {
		t.Fatal("Expected an error")
	}

	nc.Close()

	// 错误: 连接已关闭
	if err := c.PublishRequest("foo", "bar", &testdata.Person{Name: "Ivan"}); err == nil {
		t.Fatal("Expected an error")
	}

	resp := &testdata.Person{}
	// 错误: 连接已关闭
	if err := c.Request("foo", &testdata.Person{Name: "Ivan"}, resp, 2*time.Second); err == nil {
		t.Fatal("Expected an error")
	}

	// 错误: Subscription 需要 处理函数
	if _, err := c.Subscribe("foo", nil); err == nil {
		t.Fatal("Expected an error")
	}

	// 错误: 处理函数 最少需要一个参数
	if _, err := c.Subscribe("foo", func() {}); err == nil {
		t.Fatal("Expected an error")
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Expected an error")
			}
		}()
		if _, err := c.Subscribe("foo", "bar"); err == nil {
			t.Fatal("Expected an error")
		}
	}()
}

// 测试请求
func TestRequest(t *testing.T) {
	ts := RunServerOnPort(ENC_TEST_PORT)
	defer ts.Shutdown()

	dch := make(chan bool)

	opts := options
	nc, _ := opts.Connect()
	defer nc.Close()

	c, err := NewEncodedConn(nc, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		t.Fatalf("Unable to create encoded connection: %v", err)
	}
	defer c.Close()

	sentName := "Ivan"
	recvName := "Kozlovic"

	// 订阅
	if _, err := c.Subscribe("foo", func(_, reply string, p *testdata.Person) {
		if p.Name != sentName {
			t.Fatalf("Got wrong name: %v instead of %v", p.Name, sentName)
		}

		c.Publish(reply, &testdata.Person{Name: recvName})
		dch <- true
	}); err != nil {
		t.Fatalf("Unable to create subscription: %v", err)
	}

	// 订阅
	if _, err := c.Subscribe("foo", func(_ string, p *testdata.Person) {
		if p.Name != sentName {
			t.Fatalf("Got wrong name: %v instead of %v", p.Name, sentName)
		}
		dch <- true
	}); err != nil {
		t.Fatalf("Unable to create subscription: %v", err)
	}

	if err := c.Publish("foo", &testdata.Person{Name: sentName}); err != nil {
		t.Fatalf("Unable to publish: %v", err)
	}

	if err := Wait(dch); err != nil {
		t.Fatal("Did not get message")
	}
	if err := Wait(dch); err != nil {
		t.Fatal("Did not get message")
	}

	response := &testdata.Person{}
	// 请求
	if err := c.Request("foo", &testdata.Person{Name: sentName}, response, 2*time.Second); err != nil {
		t.Fatalf("Unable to publish: %v", err)
	}
	if response.Name != recvName {
		t.Fatalf("Wrong response: %v instead of %v", response.Name, recvName)
	}

	if err := Wait(dch); err != nil {
		t.Fatal("Did not get message")
	}
	if err := Wait(dch); err != nil {
		t.Fatal("Did not get message")
	}

	c2, err := NewEncodedConn(nc, GOB_ENCODER)
	if err != nil {
		t.Fatalf("Unable to create encoded connection: %v", err)
	}
	defer c2.Close()

	// 队列订阅
	if _, err := c2.QueueSubscribe("bar", "baz", func(m *Msg) {
		response := &Msg{Subject: m.Reply, Data: []byte(recvName)}
		c2.Conn.PublishMsg(response)
		dch <- true
	}); err != nil {
		t.Fatalf("Unable to create subscription: %v", err)
	}

	mReply := Msg{}

	// 请求
	if err := c2.Request("bar", &Msg{Data: []byte(sentName)}, &mReply, 2*time.Second); err != nil {
		t.Fatalf("Unable to send request: %v", err)
	}
	if string(mReply.Data) != recvName {
		t.Fatalf("Wrong reply: %v instead of %v", string(mReply.Data), recvName)
	}

	if err := Wait(dch); err != nil {
		t.Fatal("Did not get message")
	}

	if c.LastError() != nil {
		t.Fatalf("Unexpected connection error: %v", c.LastError())
	}
	if c2.LastError() != nil {
		t.Fatalf("Unexpected connection error: %v", c2.LastError())
	}
}

// 测试请求 gob
func TestRequestGOB(t *testing.T) {
	ts := RunServerOnPort(ENC_TEST_PORT)
	defer ts.Shutdown()

	type Request struct {
		Name string
	}

	type Person struct {
		Name string
		Age  int
	}

	nc, err := Connect(options.Url)
	if err != nil {
		t.Fatalf("Could not connect: %v", err)
	}
	defer nc.Close()

	ec, err := NewEncodedConn(nc, GOB_ENCODER)
	if err != nil {
		t.Fatalf("Unable to create encoded connection: %v", err)
	}
	defer ec.Close()

	// 队列订阅
	ec.QueueSubscribe("foo.request", "g", func(subject, reply string, r *Request) {
		if r.Name != "meg" {
			t.Fatalf("Expected request to be 'meg', got %q", r)
		}
		response := &Person{Name: "meg", Age: 21}
		ec.Publish(reply, response)
	})

	reply := Person{}
	// 请求
	if err := ec.Request("foo.request", &Request{Name: "meg"}, &reply, time.Second); err != nil {
		t.Fatalf("Failed to receive response: %v", err)
	}
	if reply.Name != "meg" || reply.Age != 21 {
		t.Fatalf("Did not receive proper response, %+v", reply)
	}
}
