package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
)

type eventStore struct {
	sync.RWMutex
	stream []*common.TopicEvent
}

var (
	servicePort   = os.Getenv("JOINER_SERVICE_PORT") // service port
	streamsEnv    = os.Getenv("JOINER_STREAMS_INFO") // a ;-separated list of ClusterStream|Topic|Route
	targetEnv     = os.Getenv("JOINER_TARGET")       // component[/path] to route result
	windowSizeEnv = os.Getenv("JOINER_WINDOW_SIZE")  // window size formatted as Go duration   (i.e. 1m, 3ms, etc)

	inputChan  chan *common.TopicEvent
	outputChan chan []byte

	store  *eventStore
	window *time.Ticker
)

func (s *eventStore) reset() {
	s.stream = nil
}

func main() {
	if servicePort == "" {
		servicePort = ":8080"
	}
	if streamsEnv == "" {
		log.Fatalf("joiner: evn JOINER_STREAMS_INFO not provided")
	}
	if targetEnv == "" {
		log.Fatalf("joiner: evn JOINER_TARGET not provided")
	}
	if windowSizeEnv == "" {
		windowSizeEnv = "10ms"
	}
	log.Printf("joiner: service-port: %s, stream-info: %s ==> target: %s (every %s)", servicePort, streamsEnv, targetEnv, windowSizeEnv)

	ctx := context.Background()

	// setup client
	client, err := dapr.NewClient()
	if err != nil {
		log.Fatalf("joiner: client failed: %s", err)
	}
	defer client.Close()
	targetParts, err := getTargetParts()
	if err != nil {
		log.Fatalf("joiner: target: %s", err)
	}

	// setup time window
	winDur, err := time.ParseDuration(windowSizeEnv)
	if err != nil {
		log.Fatalf("joiner: time window: %s", err)
	}
	window = time.NewTicker(winDur)
	store = &eventStore{}

	// setup internal channels for data processing
	inputChan = make(chan *common.TopicEvent)
	outputChan = make(chan []byte, 1024)
	if err := startInputLoop(ctx, window, inputChan, outputChan); err != nil {
		log.Fatalf("joiner: input loop: %s", err)
	}
	if err := startOutputLoop(ctx, client, outputChan, targetParts); err != nil {
		log.Fatalf("joiner: target invoker: %s", err)
	}

	// setup service
	svc := daprd.NewService(servicePort)
	streamsInfo := strings.Split(streamsEnv, ";")

	for _, stream := range streamsInfo {
		sub, err := getSubscription(stream)
		if err != nil {
			log.Fatalf("joiner: failed to get subscription: %s", err)
		}
		if err := svc.AddTopicEventHandler(sub, makeEventHandler(inputChan)); err != nil {
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

func makeEventHandler(inputChan chan *common.TopicEvent) common.TopicEventHandler {
	return func(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
		inputChan <- e
		return false, nil
	}
}

func getTargetParts() ([]string, error) {
	parts := strings.Split(targetEnv, "/")
	switch {
	case len(parts) > 1:
		return parts, nil
	case len(parts) == 1:
		parts = append(parts, parts[0])
		return parts, nil
	default:
		return nil, fmt.Errorf("target malformed")
	}
}

// startInputLoop reads incoming stream from eventChan and either:
//  - Store stream for aggregation, or
//  - Check time window, if closed: aggregate stored data, and
//  - Send aggregated data to outputChan for further processing
func startInputLoop(ctx context.Context, window *time.Ticker, input chan *common.TopicEvent, output chan []byte) error {
	log.Print("joiner: starting input loop")
	go func() {
		for {
			select {
			case e := <-input: // store stream while window is opened
				store.Lock()
				store.stream = append(store.stream, e)
				store.Unlock()
				logStore(store)
			case <-window.C: // aggregate stream, when window closes
				log.Println("joiner: window closed")
				jsonData, err := json.Marshal(aggregateAllEvents(store))
				if err != nil {
					log.Println("joiner: failed to marshal json data")
					continue
				}
				output <- jsonData
				store.reset()
			case <-ctx.Done():
				log.Println("joiner: event processor done!")
				break
			}
		}
	}()

	return nil
}

// startOutputLoop does the followings:
//  - Reads aggregated data (from dataChan)
//  - Encodes as json
//  - Send
func startOutputLoop(ctx context.Context, client dapr.Client, outChan chan []byte, targetParts []string) error {
	log.Print("joiner: starting output loop")
	go func() {
		for {
			select {
			case data := <-outChan:
				content := &dapr.DataContent{
					Data:        data,
					ContentType: "application/json",
				}
				appId, route := targetParts[0], targetParts[1]
				client.InvokeMethodWithContent(ctx, appId, route, http.MethodPost, content)
				log.Printf("joiner: output json data: %s", string(content.Data))
			case <-ctx.Done():
				log.Println("joiner: output invoker done!")
				break
			}
		}
	}()
	return nil
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

// aggregateAllEvents collates stored event data as large map with ID used as key:
// {"<id0>": {}, "<id1>":{},..., "<idN>":{}}
func aggregateAllEvents(store *eventStore) map[string]interface{} {
	store.RLock()
	defer store.RUnlock()
	log.Printf("joiner: aggregating %d events", len(store.stream))
	bucket := make(map[string]interface{})
	for _, event := range store.stream {
		bucket[event.ID] = event.Data
	}
	return bucket
}

func logStore(store *eventStore) {
	store.RLock()
	defer store.RUnlock()
	for _, event := range store.stream {
		log.Printf("joiner: stored event: {id: %s : {topic: %s; data: %s}}\n", event.ID, event.Topic, string(event.RawData))
	}
}