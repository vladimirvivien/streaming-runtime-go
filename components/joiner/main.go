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
	"github.com/google/cel-go/cel"
	"github.com/vladimirvivien/streaming-runtime/components/support"
)

type eventStore struct {
	sync.RWMutex
	streams map[string][]*common.TopicEvent
}

var (
	servicePort   = os.Getenv("JOINER_SERVICE_PORT")             // service port
	stream0Env    = os.Getenv("JOINER_STREAM0_INFO")             // a |-separated list of info for stream 0
	stream1Env    = os.Getenv("JOINER_STREAM1_INFO")             // a |-separated list of info for stream 1
	topics        []string                                       // names of known topics
	targetEnv     = os.Getenv("JOINER_TARGET")                   // component[/path] to route result
	windowSizeEnv = os.Getenv("JOINER_WINDOW_SIZE")              // window size formatted as Go duration   (i.e. 1m, 3ms, etc)
	filterExprEnv = os.Getenv("JOINER_SELECT_FILTER_EXPRESSION") // expression used to filter data from stream
	dataExprEnv   = os.Getenv("JOINER_SELECT_DATA_EXPRESSION")   // expression used to generate data output from streams

	inputChan  chan *common.TopicEvent
	outputChan chan []byte

	store  *eventStore
	window *time.Ticker

	filterProg cel.Program
	dataProg   cel.Program
)

func (s *eventStore) reset() {
	s.Lock()
	defer s.Unlock()
	for _, topic := range topics {
		if s.streams[topic] != nil {
			s.streams[topic] = nil
		}
	}
	s.streams = nil
	s.streams = make(map[string][]*common.TopicEvent)
}

func main() {
	if servicePort == "" {
		servicePort = ":8080"
	}
	if stream0Env == "" || stream1Env == "" {
		log.Fatalf("joiner: env JOINER_STREAM0_INFO or JOINER_STREAM1_INFO missing")
	}
	if targetEnv == "" {
		log.Fatalf("joiner: env JOINER_TARGET not provided")
	}
	if windowSizeEnv == "" {
		windowSizeEnv = "10ms"
	}
	log.Printf("joiner: service-port: %s, streams: (%s;%s) filter: (%s) ==> target: %s (every %s)",
		servicePort, stream0Env, stream1Env, filterExprEnv, targetEnv, windowSizeEnv)
	// setup internal channels for data processing
	inputChan = make(chan *common.TopicEvent)
	outputChan = make(chan []byte, 1024)

	ctx := context.Background()

	// setup client
	client, err := dapr.NewClient()
	if err != nil {
		log.Fatalf("joiner: client failed: %s", err)
	}
	defer client.Close()
	targetParts, err := support.GetTargetParts(targetEnv)
	if err != nil {
		log.Fatalf("joiner: target: %s", err)
	}

	// setup time window
	winDur, err := time.ParseDuration(windowSizeEnv)
	if err != nil {
		log.Fatalf("joiner: time window: %s", err)
	}
	window = time.NewTicker(winDur)
	store = &eventStore{streams: make(map[string][]*common.TopicEvent)}

	// setup service handlers
	svc := daprd.NewService(servicePort)
	streamsInfo := []string{stream0Env, stream1Env}

	// setup topic handler for each subscription
	for _, stream := range streamsInfo {
		sub, err := getSubscription(stream)
		if err != nil {
			log.Fatalf("joiner: failed to get subscription: %s", err)
		}
		topics = append(topics, sub.Topic)
		store.streams[sub.Topic] = nil

		if err := svc.AddTopicEventHandler(sub, makeEventHandler(inputChan)); err != nil {
			log.Fatalf("joiner: pubsub: %s: failed: %s", sub.PubsubName, err)
		}
	}

	// setup common expression lang (cel) programs
	// for data selection and data filtering
	if filterExprEnv != "" {
		prog, err := support.CompileCELProg(filterExprEnv, topics...)
		if err != nil {
			log.Fatalf("joiner: filter expression: %s", err)
		}
		filterProg = prog
	}
	if dataExprEnv != "" {
		prog, err := support.CompileCELProg(dataExprEnv, topics...)
		if err != nil {
			log.Fatalf("joiner: data selection expression: %s", err)
		}
		dataProg = prog
	}

	// setup event processors
	if err := startInputLoop(ctx, window, inputChan, outputChan); err != nil {
		log.Fatalf("joiner: input loop: %s", err)
	}
	if err := startOutputLoop(ctx, client, outputChan, targetParts); err != nil {
		log.Fatalf("joiner: target invoker: %s", err)
	}

	// start dapr services
	log.Println("joiner: starting on port ", servicePort)
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
				store.streams[e.Topic] = append(store.streams[e.Topic], e)
				store.Unlock()
			case <-window.C: // aggregate stream, when window closes
				events, err := aggregateEvents(store)
				if err != nil {
					log.Printf("joiner: failed to aggregate events: %s", err)
					continue
				}
				jsonData, err := json.Marshal(events)
				if err != nil {
					log.Println("joiner: failed to marshal json data")
					continue
				}
				output <- jsonData
				store.reset()
			case <-ctx.Done():
				log.Println("joiner: event processor done!")
				return
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

// aggregatedEvents applies left join semantics to select and filter data
func aggregateEvents(store *eventStore) (interface{}, error) {
	store.RLock()
	defer store.RUnlock()

	var bucket []interface{}
	topicA, topicB := topics[0], topics[1]
	for _, eventA := range store.streams[topicA] {
		for _, eventB := range store.streams[topicB] {
			// TODO 1) apply filter expression 2) if ok, apply join expression 3) send to output
			shouldCollect, err := shouldCollect(eventA, eventB, filterProg)
			if err != nil {
				return nil, err
			}
			if shouldCollect {
				data, err := collectData(eventA, eventB, dataProg)
				if err != nil {
					return nil, err
				}
				bucket = append(bucket, data)
			}
		}
	}

	return bucket, nil
}

func collectData(eventA, eventB *common.TopicEvent, prog cel.Program) (interface{}, error) {
	dataMap := map[string]interface{}{
		eventA.Topic: eventA.Data,
		eventB.Topic: eventB.Data,
	}
	if prog != nil {
		result, _, err := prog.Eval(dataMap)
		if err != nil {
			return nil, err
		}
		return result.Value(), nil
	}
	return dataMap, nil
}

func shouldCollect(eventA, eventB *common.TopicEvent, prog cel.Program) (bool, error) {
	if filterProg != nil {
		dataMap := map[string]interface{}{
			eventA.Topic: eventA.Data,
			eventB.Topic: eventB.Data,
		}
		filterResult, _, err := prog.Eval(dataMap)
		if err != nil {
			return false, err
		}
		if filterResult.Type().TypeName() != "bool" {
			return false, fmt.Errorf("select: filter expression: must return a boolean")
		}
		shouldCollect := filterResult.Value().(bool)
		return shouldCollect, nil
	} else {
		return true, nil // always collect if no program provided.
	}
}

func logStore(store *eventStore) {
	store.RLock()
	defer store.RUnlock()
	for _, topic := range topics {
		for _, event := range store.streams[topic] {
			log.Printf("joiner: stored event: {id: %s : {topic: %s; data: %s}}\n", event.ID, event.Topic, string(event.RawData))
		}
	}
}
