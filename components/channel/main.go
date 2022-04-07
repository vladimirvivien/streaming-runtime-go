package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/vladimirvivien/streaming-runtime/components/support"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	servicePort          = os.Getenv("CHANNEL_SERVICE_PORT")        // service port
	modeEnv              = os.Getenv("CHANNEL_MODE")                // channel mode, valid values = {stream | aggregate}
	streamFromEnv        = os.Getenv("CHANNEL_STREAM_FROM")         // name of topic or service route to stream from
	streamToStreamEnv    = os.Getenv("CHANNEL_STREAM_TO_STREAM")    // "pubsub/topic" path where to route result
	streamToComponentEnv = os.Getenv("CHANNEL_STREAM_TO_COMPONENT") // "component/route" path where to send result
	streamFilterExprEnv  = os.Getenv("CHANNEL_STREAM_WHERE")        // expression used to filter data from stream
	streamSelectExprEnv  = os.Getenv("CHANNEL_STREAM_SELECT")       // expression used to generate data output from streams

	triggerExprEnv = os.Getenv("CHANNEL_AGGREGATE_TRIGGER") // expression to trigger aggregation
	counter        = 0

	inputChan  chan *common.InvocationEvent
	outputChan chan []byte

	filterProg  cel.Program
	dataProg    cel.Program
	triggerProg cel.Program
)

func main() {
	if servicePort == "" {
		servicePort = ":8080"
	}
	if streamFromEnv == "" {
		streamFromEnv = support.SanitizeIdentifier(os.Getenv("APP_ID"))
	}
	if modeEnv == "" {
		modeEnv = "stream"
	}
	if streamToStreamEnv == "" && streamToComponentEnv == "" {
		log.Fatalf("channel: env CHANNEL_TARGET_STREAM and/or CHANNEL_TARGET_COMPONENT must be provided")
	}

	log.Printf("channel: service-port: %s [stream-source=%s], filterExpr: (%s), dataExpr: (%s), mode: %s ==> target: stream(%s) component(%s)",
		servicePort, streamFromEnv, streamFilterExprEnv, streamSelectExprEnv, modeEnv, streamToStreamEnv, streamToComponentEnv)

	// setup internal channels for data processing
	inputChan = make(chan *common.InvocationEvent, 1024)
	outputChan = make(chan []byte, 1024)

	ctx := context.Background()

	// setup client
	client, err := dapr.NewClient()
	if err != nil {
		log.Fatalf("channel: client failed: %s", err)
	}
	defer client.Close()

	targetStreamParts, err := support.GetTargetParts(streamToStreamEnv)
	if err != nil {
		log.Fatalf("channel: stream target: %s", err)
	}
	targetComponentParts, err := support.GetTargetParts(streamToComponentEnv)
	if err != nil {
		log.Fatalf("channel: component target: %s", err)
	}

	// start service
	svc := daprd.NewService(servicePort)
	if err := svc.AddServiceInvocationHandler(streamFromEnv, invocationHandler); err != nil {
		log.Fatalf("channel: service route: %s: failed: %s", streamFromEnv, err)
	}

	// setup common expression lang (cel) programs
	// for data selection and data filtering
	if streamFilterExprEnv != "" {
		prog, err := support.CompileCELProg(streamFilterExprEnv, decls.NewVar(streamFromEnv, decls.NewMapType(decls.String, decls.Dyn)))
		if err != nil {
			log.Fatalf("channel: filter expression: %s", err)
		}
		filterProg = prog
	}
	if streamSelectExprEnv != "" {
		prog, err := support.CompileCELProg(streamSelectExprEnv, decls.NewVar(streamFromEnv, decls.NewMapType(decls.String, decls.Dyn)))
		if err != nil {
			log.Fatalf("channel: data selection expression: %s", err)
		}
		dataProg = prog
	}
	if triggerExprEnv != "" {
		prog, err := support.CompileCELProg(triggerExprEnv,
			decls.NewVar(streamFromEnv, decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("count", decls.Int),
			decls.NewVar("duration", decls.String),
		)
		if err != nil {
			log.Fatalf("channel: data selection expression: %s", err)
		}
		dataProg = prog
	}

	// setup event processors
	if err := startProcessingLoop(ctx, inputChan, outputChan); err != nil {
		log.Fatalf("channel: input loop: %s", err)
	}
	if err := startOutputLoop(ctx, client, outputChan, targetStreamParts, targetComponentParts); err != nil {
		log.Fatalf("channel: ouptut loop: %s", err)
	}

	// start dapr services
	log.Println("channel: starting on port ", servicePort)
	if err := svc.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("channel: starting failed: %v", err)
	}
}

func invocationHandler(ctx context.Context, e *common.InvocationEvent) (out *common.Content, err error) {
	log.Printf("event received: content-type: %s, content-url: %s, qury: %s data(%s) ", e.ContentType, e.DataTypeURL, e.QueryString, string(e.Data))
	inputChan <- e
	return &common.Content{
		Data:        e.Data,
		ContentType: e.ContentType,
		DataTypeURL: e.DataTypeURL,
	}, nil
}

func startProcessingLoop(ctx context.Context, input chan *common.InvocationEvent, output chan []byte) error {
	go func() {
		for {
			select {
			case data := <-input:
				shouldCollect, err := shouldCollect(data, filterProg)
				if err != nil {
					log.Printf("channel: %s", err)
					continue
				}
				if shouldCollect {
					event, err := collectData(data, dataProg)
					if err != nil {
						log.Printf("channel: event collection: %s", err)
					}

					counter++
					output <- event
				}
			case <-ctx.Done():
				log.Println("channel: input channel shutdown")
				break
			}
		}
	}()

	return nil
}

func startOutputLoop(ctx context.Context, client dapr.Client, output chan []byte, targetStreamParts, targetComponentParts []string) error {
	go func() {
		for {
			select {
			case data := <-output:
				shouldTrigger, err := shouldTrigger(triggerProg)
				if err != nil {
					log.Printf("channel: should trigger: %s", err)
					continue
				}
				if shouldTrigger {
					if len(targetStreamParts) > 0 {
						pubsub, topic := targetStreamParts[0], targetStreamParts[1]
						if err := client.PublishEvent(ctx, pubsub, topic, data, dapr.PublishEventWithContentType("application/json")); err != nil {
							log.Printf("channel: target pubsub/stream: %s", err)
						} else {
							log.Printf("channel: target pubsub/stream: output: %s", string(data))
						}
					}

					if len(targetComponentParts) > 0 {
						content := &dapr.DataContent{
							Data:        data,
							ContentType: "application/json",
						}
						componentId, route := targetComponentParts[0], targetComponentParts[1]
						if _, err := client.InvokeMethodWithContent(ctx, componentId, route, http.MethodPost, content); err != nil {
							log.Printf("channel: target component service: %s", err)
						} else {
							log.Printf("channel: target component service: output: %s", string(content.Data))
						}
					}
				}
			case <-ctx.Done():
				log.Println("channel: output loop shutting down!")
				break
			}
		}
	}()
	return nil
}

// shouldCollect applies filtering expression (if any) to determine if that
// event should be collected for downstream propagation
func shouldCollect(event *common.InvocationEvent, prog cel.Program) (bool, error) {
	if prog != nil {
		jsonData, err := support.ExtractJSONFromInvocation(event)
		if err != nil {
			return false, fmt.Errorf("filter expression: marshal data: %s", err)
		}
		dataMap := map[string]interface{}{
			streamFromEnv: jsonData,
		}

		filterResult, _, err := prog.Eval(dataMap)
		if err != nil {
			return false, fmt.Errorf("filter expression: evaluation:%s", err)
		}
		if filterResult.Type().TypeName() != "bool" {
			return false, fmt.Errorf("filter expression: must return a boolean")
		}
		shouldCollect := filterResult.Value().(bool)
		return shouldCollect, nil
	}
	return true, nil // always collect if no program provided.
}

// shouldTrigger returns true if the trigger expression (time duration or
// event count) evaluates to true or if the expression is not provided.
func shouldTrigger(prog cel.Program) (bool, error) {
	//if prog != nil {
	//	dataMap := map[string]interface{}{
	//		"count": counter,
	//	}
	//
	//}
	return true, nil
}

// collectData applies data collection expression (if any) and returns
// the collected event (original or synthetic) for downstream propagation.
func collectData(event *common.InvocationEvent, prog cel.Program) ([]byte, error) {
	if prog != nil {
		data, err := support.ExtractJSONFromInvocation(event)
		if err != nil {
			return nil, fmt.Errorf("data collection: marshal data: %s", err)
		}
		dataMap := map[string]interface{}{
			streamFromEnv: data,
		}

		result, _, err := prog.Eval(dataMap)
		if err != nil {
			return nil, fmt.Errorf("data collection: failed to evaluate filter expression: %s", err)
		}
		conv, err := result.ConvertToNative(reflect.TypeOf(&structpb.Struct{}))
		if err != nil {
			return nil, fmt.Errorf("data collection: failed to convert to native: %s", err)
		}
		jsonData, err := conv.(*structpb.Struct).MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("data collection: failed marshal to JSON: %s", err)
		}
		return jsonData, nil
	}

	return event.Data, nil
}
