package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
)

func main() {
	svc := daprd.NewService(":8080")

	// exposes endpoint to receive message events
	if err := svc.AddServiceInvocationHandler("/messages", messageHandler); err != nil {
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
	log.Printf("/messages invoked: [content-type: %s, url: %s?%s, data: %s", in.ContentType, in.DataTypeURL, in.QueryString , string(in.Data))
	return &common.Content{
		Data:        in.Data,
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}, nil
}