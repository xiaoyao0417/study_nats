# Request 使用
按主题发送请求,等待返回消息。
没有正确收到消息，或者错误，或者超时

1. 启动nats
docker-compose up

2. 启动请求
 go run main.go test "this is a request msg"
