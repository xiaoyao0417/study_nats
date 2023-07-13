package natsdeal

import (
	"go-nats-jetstream-example/config"
	"log"

	"github.com/nats-io/nats.go"
)

// 初始化
func JetStreamInit() (nats.JetStreamContext, error) {
	// Connect to NATS
	// nc, err := nats.Connect(nats.DefaultURL)

	nc, err := nats.Connect("13.250.90.207:4222")
	if err != nil {
		return nil, err
	}

	// Create JetStream Context
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		return nil, err
	}

	// Create a stream if it does not exist
	err = CreateStream(js)
	if err != nil {
		return nil, err
	}

	return js, nil
}

func CreateStream(jetStream nats.JetStreamContext) error {
	stream, err := jetStream.StreamInfo(config.StreamName)

	// stream not found, create it
	if stream == nil {
		log.Printf("Creating stream: %s\n", config.StreamName)

		_, err = jetStream.AddStream(&nats.StreamConfig{
			Name:     config.StreamName,
			Subjects: []string{config.StreamSubjects},
		})

		if err != nil {
			return err
		}
	}
	return nil
}
