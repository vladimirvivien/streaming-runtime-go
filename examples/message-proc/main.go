package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
)

var (
	servicePort = os.Getenv("PROC_SERVICE_PORT")
	serviceRoute = os.Getenv("PROC_SERVICE_ROUTE")
)

func main() {
	if servicePort == "" {
		servicePort = ":8080"
	}
	if serviceRoute == "" {
		serviceRoute = os.Getenv("APP_ID")
	}

	svc := daprd.NewService(servicePort)


	// exposes endpoint to receive message events
	if err := svc.AddServiceInvocationHandler(serviceRoute, messageHandler); err != nil {
		log.Fatalf("Failed to add orders handler: %s", err)
	}

	log.Println("Starting service app, port 8080")

	// call Service.Start to start listening to incoming requests
	if err := svc.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("error listenning: %v", err)
	}
}

func messageHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	if in == nil {
		return nil, fmt.Errorf("invocation parameter required")
	}
	log.Printf("Stream received: %s", string(in.Data))
	return &common.Content{
		Data:        in.Data,
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}, nil
}