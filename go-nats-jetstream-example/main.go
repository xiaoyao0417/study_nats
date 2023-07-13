package main

import (
	"go-nats-jetstream-example/natsdeal"
	"log"
	"sync"
)

func main() {
	log.Println("Starting...")

	js, err := natsdeal.JetStreamInit()
	if err != nil {
		log.Println(err)
		return
	}

	// Let's assume that publisher and consumer are services running on different servers.
	// So run publisher and consumer asynchronously to see how it works
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		natsdeal.PublishReviews(js)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		natsdeal.ConsumeReviews(js)
	}()

	wg.Wait()

	log.Println("Exit...")
}
