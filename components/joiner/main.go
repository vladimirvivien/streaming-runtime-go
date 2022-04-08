package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	commontypes "github.com/google/cel-go/common/types/ref"
	"github.com/vladimirvivien/streaming-runtime/components/support"
	exprv1alpha1 "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/types/known/structpb"
)

type eventStore struct {
	sync.RWMutex
	streams map[string][]*common.TopicEvent
}

var (
	servicePort          = os.Getenv("JOINER_SERVICE_PORT")        // service port
	streamFrom0Env       = os.Getenv("JOINER_STREAM_FROM_0")       // a |-separated list of info for stream 0
	streamFrom1Env       = os.Getenv("JOINER_STREAM_FROM_1")       // a |-separated list of info for stream 1
	streamToComponentEnv = os.Getenv("JOINER_STREAM_TO_COMPONENT") // component[/path] to route result
	streamToStreamEnv    = os.Getenv("JOINER_STREAM_TO_STREAM")    // pubsub[/topic] where to route result
	streamFilterExprEnv  = os.Getenv("JOINER_STREAM_FILTER")       // expression used to filter data from stream
	streamSelectExprEnv  = os.Getenv("JOINER_STREAM_SELECT")       // expression used to generate data output from streams
	windowSizeEnv        = os.Getenv("JOINER_WINDOW_SIZE")         // window size formatted as Go duration   (i.e. 1m, 3ms, etc)
	topics               []string                                  // names of known topics

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
	if streamFrom0Env == "" || streamFrom1Env == "" {
		log.Fatalf("joiner: env JOINER_STREAM_FROM_0 or JOINER_STREAM_FROM_1 missing")
	}
	if streamToComponentEnv == "" {
		log.Fatalf("joiner: env JOINER_STREAM_TO_COMPONENT not provided")
	}
	if windowSizeEnv == "" {
		windowSizeEnv = "10ms"
	}
	log.Printf("joiner: service-port: %s, streams: (%s;%s) filter: (%s) ==> target: %s (every %s)",
		servicePort, streamFrom0Env, streamFrom1Env, streamFilterExprEnv, streamToComponentEnv, windowSizeEnv)
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

	targetComponentParts, err := support.GetTargetParts(streamToComponentEnv)
	if err != nil {
		log.Fatalf("joiner: stream.To component: %s", err)
	}
	targetStreamParts, err := support.GetTargetParts(streamToStreamEnv)
	if err != nil {
		log.Fatalf("joiner: stream.To stream: %s", err)
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
	streamsInfo := []string{streamFrom0Env, streamFrom1Env}

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

	// CEL program variables
	variables := []*exprv1alpha1.Decl{
		decls.NewVar(topics[0], decls.NewMapType(decls.String, decls.Dyn)),
		decls.NewVar(topics[1], decls.NewMapType(decls.String, decls.Dyn)),
	}

	// setup common expression lang (cel) programs
	// for data selection and data filtering
	if streamFilterExprEnv != "" {

		prog, err := support.CompileCELProg(streamFilterExprEnv, variables...)
		if err != nil {
			log.Fatalf("joiner: filter expression: %s", err)
		}
		filterProg = prog
	}
	if streamSelectExprEnv != "" {
		prog, err := support.CompileCELProg(streamSelectExprEnv, variables...)
		if err != nil {
			log.Fatalf("joiner: data selection expression: %s", err)
		}
		dataProg = prog
	}

	// setup event processors
	if err := startInputLoop(ctx, window, inputChan, outputChan); err != nil {
		log.Fatalf("joiner: input loop: %s", err)
	}
	if err := startOutputLoop(ctx, client, outputChan, targetComponentParts, targetStreamParts); err != nil {
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
				log.Printf("joiner: received data: topic=%s, data=%v", e.Topic, e.Data)
				store.streams[e.Topic] = append(store.streams[e.Topic], e)
				store.Unlock()
			case <-window.C: // aggregate stream, when window closes
				events, err := aggregateEvents(store)
				if err != nil {
					log.Printf("joiner: failed to aggregate events: %s", err)
					continue
				}
				jsonData, err := events.MarshalJSON()
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
func startOutputLoop(ctx context.Context, client dapr.Client, outChan chan []byte, targetComponentParts, targetStreamParts []string) error {
	log.Print("joiner: starting output loop")
	go func() {
		for {
			select {
			case data := <-outChan:
				if len(data) == 0 {
					log.Print("joiner: data output is zero")
					continue
				}

				if len(targetStreamParts) > 0 {
					pubsub, topic := targetStreamParts[0], targetStreamParts[1]
					if err := client.PublishEvent(ctx, pubsub, topic, data, dapr.PublishEventWithContentType("application/json")); err != nil {
						log.Printf("joiner: target pubsub/stream: %s", err)
					} else {
						log.Printf("joiner: data sent pubsub/stream %s: %s", targetStreamParts, string(data))
					}
				}

				if len(targetComponentParts) > 0 {
					content := &dapr.DataContent{
						Data:        data,
						ContentType: "application/json",
					}
					componentId, route := targetComponentParts[0], targetComponentParts[1]
					if _, err := client.InvokeMethodWithContent(ctx, componentId, route, http.MethodPost, content); err != nil {
						log.Printf("joiner: target component service: %s", err)
					} else {
						log.Printf("joiner: data sent to %s: %s", targetComponentParts, string(content.Data))
					}
				}
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
func aggregateEvents(store *eventStore) (*structpb.ListValue, error) {
	store.RLock()
	defer store.RUnlock()

	var bucket []interface{}
	topicA, topicB := topics[0], topics[1]
	if len(store.streams[topicA]) == 0 || len(store.streams[topicB]) == 0 {
		return nil, fmt.Errorf("empty stream(s): join will be empty")
	}
	for _, eventA := range store.streams[topicA] {
		for _, eventB := range store.streams[topicB] {
			// 1) apply filter expression 2) if ok, apply data join expression 3) send to output
			shouldCollect, err := shouldCollect(eventA, eventB, filterProg)
			if err != nil {
				return nil, fmt.Errorf("shouldCollect check failed: %s", err)
			}
			if shouldCollect {
				data, err := collectData(eventA, eventB, dataProg)
				if err != nil {
					return nil, fmt.Errorf("failed to collect: %s", err)
				}
				bucket = append(bucket, data.AsMap())
			}
		}
	}

	if len(bucket) == 0 {
		return nil, fmt.Errorf("join result is empty")
	}

	list, err := structpb.NewList(bucket)
	if err != nil {
		return nil, fmt.Errorf("aggregation bucket failed : %s", err)
	}
	return list, nil
}

func collectData(eventA, eventB *common.TopicEvent, prog cel.Program) (*structpb.Struct, error) {
	dataMap := map[string]interface{}{
		eventA.Topic: eventA.Data,
		eventB.Topic: eventB.Data,
	}
	if prog != nil {
		result, _, err := prog.Eval(dataMap)
		if err != nil {
			return nil, err
		}
		conv, err := result.ConvertToNative(reflect.TypeOf(&structpb.Struct{}))
		if err != nil {
			return nil, fmt.Errorf("failed to convert to native: %s", err)
		}
		return conv.(*structpb.Struct), nil
	}

	result, err := structpb.NewStruct(dataMap)
	if err != nil {
		return nil, fmt.Errorf("new structpb Value failed: %s", err)
	}
	return result, nil
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

func marshalJSON(value commontypes.Val) ([]byte, error) {
	conv, err := value.ConvertToNative(reflect.TypeOf(&structpb.Value{}))
	if err != nil {
		return nil, err
	}
	jsonData, err := conv.(*structpb.Value).MarshalJSON()
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}
