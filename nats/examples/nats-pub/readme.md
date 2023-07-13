# Publish 和 PublishRequest 使用

1. 启动nats
docker-compose up

2. 启动发布
 go run main.go test "this's a msg"

1. 问题
主题开没有开的话，是不是消息就丢失了，消息会怎么处理
先发布后开订阅,消息是不是就丢失了