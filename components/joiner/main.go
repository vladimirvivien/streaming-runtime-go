package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
)

var(
	servicePort = os.Getenv("JOINER_SERVICE_PORT")
	streamPathsEnv = os.Getenv("JOINER_STREAM_PATHS")
)

func main() {
	if servicePort == "" {
		servicePort = ":8080"
	}

	if streamPathsEnv == ""  {
		log.Fatalf("joiner: missing stream paths")
	}

	svc := daprd.NewService(servicePort)

	streamPaths := strings.Split(streamPathsEnv, ",")

	for _, path := range streamPaths {
		if err := svc.AddServiceInvocationHandler(path, streamHandler); err != nil {
			log.Fatalf("joiner: '%s' path handler failed: %s", path, err)
		}
	}

	log.Println("joiner: starting on port ", servicePort)

	// call Service.Start to start listening to incoming requests
	if err := svc.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("joiner: starting failed: %v", err)
	}

	log.Println("joiner: started on port ", servicePort)
}

func streamHandler(_ context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	if in == nil {
		return nil, fmt.Errorf("invocation parameter required")
	}
	log.Printf("stream handler invoked: [content-type: %s, url: %s?%s, data: %s", in.ContentType, in.DataTypeURL, in.QueryString , string(in.Data))
	return &common.Content{
		Data:        in.Data,
		ContentType: in.ContentType,
		DataTypeURL: in.DataTypeURL,
	}, nil
}