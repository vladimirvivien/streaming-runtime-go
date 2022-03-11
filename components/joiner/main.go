package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/vladimirvivien/streaming-runtime/components/k8s"
)

var (
	servicePort = os.Getenv("JOINER_SERVICE_PORT")
	streamsEnv  = os.Getenv("JOINER_STREAMS")
	namespace   = os.Getenv("JOINER_NAMESPACE")
)

func main() {
	if servicePort == "" {
		servicePort = ":8080"
	}
	if namespace == "" {
		namespace = "default"
	}
	if streamsEnv == "" {
		log.Fatalf("joiner: evn JOINER_STREAMS not provided")
	}
	k8sC, err := k8s.NewClient(namespace)
	if err != nil {
		log.Fatalf("joiner: failed to create Kubernetes client: %s", err)
	}

	svc := daprd.NewService(servicePort)
	streams := strings.Split(streamsEnv, ",")

	for _, stream := range streams {
		sub, err := getSubscription(k8sC, stream)
		if err != nil {
			log.Fatalf("joiner: failed retrieve subscription object: %s", err)
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

func getSubscription(k8sC *k8s.Client, subName string) (*common.Subscription, error) {
	sub, err := k8sC.GetSubscription(context.Background(), subName)
	if err != nil {
		return nil, err
	}
	return &common.Subscription{
		PubsubName: sub.Spec.Pubsubname,
		Topic:      sub.Spec.Topic,
		Metadata:   nil,
		Route:      sub.Spec.Routes.Default,
	}, nil
}
