package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	dapr "github.com/dapr/go-sdk/client"
)

var(
	pubsubName = os.Getenv("CLUSTER_STREAM")
	topicName = os.Getenv("STREAM_TOPIC")
)

func main() {
	client, err := dapr.NewClient()
	if err != nil {
		log.Printf("Failed to create Dapr client: %s",err)
		os.Exit(1)
	}

	log.Printf("Message-gen client created: clusterStream: %s, topic: %s", pubsubName, topicName)

	defer client.Close()
	ctx := context.Background()

	n := 0
	for {
		n++
		message := fmt.Sprintf(`{"message": {"id": %d, "text":"time is %s"}}`, n, time.Now().String())
		if err := client.PublishEvent(ctx, pubsubName, topicName, message, dapr.PublishEventWithContentType("application/json")); err != nil {
			panic(err)
		}
		time.Sleep(time.Second*7)
	}
}

