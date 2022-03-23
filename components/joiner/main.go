package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
)

var (
	servicePort = os.Getenv("JOINER_SERVICE_PORT")
	// JOINER_STREAMS_INFO: a ;-separated list of ClusterStream|Topic|Route
	streamsEnv = os.Getenv("JOINER_STREAMS_INFO")
	namespace  = os.Getenv("JOINER_NAMESPACE")
)

func main() {
	if servicePort == "" {
		servicePort = ":8080"
	}
	log.Printf("joiner: JOINER_SERVICE_PORT: %s", servicePort)
	if namespace == "" {
		namespace = "default"
	}
	log.Printf("joiner: JOINER_NAMESPACE: %s", namespace)
	if streamsEnv == "" {
		log.Fatalf("joiner: evn JOINER_STREAMS_INFO not provided")
	}
	log.Printf("joiner: JOINER_STREAMS_INFO: %s", streamsEnv)

	//k8sC, err := k8s.NewClient(namespace)
	//if err != nil {
	//	log.Fatalf("joiner: failed to create Kubernetes client: %s", err)
	//}

	svc := daprd.NewService(servicePort)
	streamsInfo := strings.Split(streamsEnv, ";")

	for _, stream := range streamsInfo {
		sub, err := getSubscription(stream)
		if err != nil {
			log.Fatalf("joiner: failed to get subscription: %s", err)
		}
		if err := svc.AddTopicEventHandler(sub, makeHandler(sub.Route)); err != nil {
			log.Fatalf("joiner: pubsub: %s: failed: %s", sub.PubsubName, err)
		}
	}

	log.Println("joiner: starting on port ", servicePort)

	// call Service.Start to start listening to incoming requests
	if err := svc.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("joiner: starting failed: %v", err)
	}

	log.Println("joiner: started on port ", servicePort)
}

func makeHandler(path string) common.TopicEventHandler {

	return func(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
		log.Printf("event: [ID: %s, Type: %s; Pubsub: %s; Topic: %s]", e.ID, e.Type, e.PubsubName, e.Topic)
		return false, nil
	}

}

func getSubscription(streamInfo string) (*common.Subscription, error) {
	streamPart := strings.Split(streamInfo, "|")

	return &common.Subscription{
		PubsubName: streamPart[0],
		Topic:      streamPart[1],
		Metadata:   nil,
		Route:      streamPart[2],
	}, nil
}
