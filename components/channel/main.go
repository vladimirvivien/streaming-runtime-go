package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/google/cel-go/cel"
	"github.com/vladimirvivien/streaming-runtime/components/support"
)

var (
	servicePort   = os.Getenv("CHANNEL_SERVICE_PORT")             // service port
	serviceRoute  = os.Getenv("CHANNEL_SERVICE_ROUTE")            // service path
	targetEnv     = os.Getenv("CHANNEL_TARGET")                   // component[/path] to route result
	filterExprEnv = os.Getenv("CHANNEL_SELECT_FILTER_EXPRESSION") // expression used to filter data from stream
	dataExprEnv   = os.Getenv("CHANNEL_SELECT_DATA_EXPRESSION")   // expression used to generate data output from streams

	inputChan  chan map[string]interface{}
	outputChan chan []byte

	filterProg cel.Program
	dataProg   cel.Program
)

func main() {
	if servicePort == "" {
		servicePort = ":8080"
	}
	if serviceRoute == "" {
		serviceRoute = "/default"
	}
	if targetEnv == "" {
		log.Fatalf("channel: env CHANNEL_TARGET not provided")
	}
	log.Printf("channel: service-port: %s[/%s], filterExpr: (%s), dataExpr: (%s) ==> target: %s",
		servicePort, serviceRoute, filterExprEnv, dataExprEnv, targetEnv)
	// setup internal channels for data processing
	inputChan = make(chan map[string]interface{}, 1024)
	outputChan = make(chan []byte, 1024)

	ctx := context.Background()

	// setup client
	client, err := dapr.NewClient()
	if err != nil {
		log.Fatalf("channel: client failed: %s", err)
	}
	defer client.Close()

	targetParts, err := support.GetTargetParts(targetEnv)
	if err != nil {
		log.Fatalf("channel: target: %s", err)
	}

	// start service
	svc := daprd.NewService(servicePort)
	if err := svc.AddServiceInvocationHandler(serviceRoute, invocationHandler); err != nil {
		log.Fatalf("channel: service route: %s: failed: %s", serviceRoute, err)
	}

	// setup common expression lang (cel) programs
	// for data selection and data filtering
	if filterExprEnv != "" {
		prog, err := support.CompileCELProg(filterExprEnv, "event")
		if err != nil {
			log.Fatalf("channel: filter expression: %s", err)
		}
		filterProg = prog
	}
	if dataExprEnv != "" {
		prog, err := support.CompileCELProg(dataExprEnv, "event")
		if err != nil {
			log.Fatalf("channel: data selection expression: %s", err)
		}
		dataProg = prog
	}

	// setup event processors
	if err := startProcessingLoop(ctx, inputChan, outputChan); err != nil {
		log.Fatalf("channel: input loop: %s", err)
	}
	if err := startOutputLoop(ctx, client, outputChan, targetParts); err != nil {
		log.Fatalf("channel: target invoker: %s", err)
	}

	// start dapr services
	log.Println("channel: starting on port ", servicePort)
	if err := svc.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("channel: starting failed: %v", err)
	}

	log.Println("channel: started on port ", servicePort)
}

func invocationHandler(ctx context.Context, e *common.InvocationEvent) (out *common.Content, err error) {
	if e.ContentType != "application/json" {
		return nil, fmt.Errorf("channel: content type not supported: %s", e.ContentType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return nil, fmt.Errorf("channel: JSON unmarshal: %w", err)
	}
	inputChan <- data
	return &common.Content{
		Data:        e.Data,
		ContentType: e.ContentType,
		DataTypeURL: e.DataTypeURL,
	}, nil
}

func startProcessingLoop(ctx context.Context, input chan map[string]interface{}, output chan []byte) error {
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
					d, err := collectData(data, dataProg)
					if err != nil {
						log.Printf("channel: %s", err)
					}
					output <- d
				}
			case <-ctx.Done():
				log.Println("channel: input channel shutdown")
				break
			}
		}
	}()

	return nil
}

func startOutputLoop(ctx context.Context, client dapr.Client, output chan []byte, targetParts []string) error {
	go func() {
		for {
			select {
			case data := <-output:
				content := &dapr.DataContent{
					Data:        data,
					ContentType: "application/json",
				}
				appId, route := targetParts[0], targetParts[1]
				client.InvokeMethodWithContent(ctx, appId, route, http.MethodPost, content)
				log.Printf("channel: output json data: %s", string(content.Data))
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
func shouldCollect(event map[string]interface{}, prog cel.Program) (bool, error) {
	if prog != nil {
		dataMap := map[string]interface{}{
			"event": event,
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

// collectData applies data collection expression (if any) and returns
// the event as []byte for downstream propagation
func collectData(event map[string]interface{}, prog cel.Program) ([]byte, error) {
	dataMap := map[string]interface{}{
		"event": event,
	}
	var value interface{}
	if prog != nil {
		result, _, err := prog.Eval(dataMap)
		if err != nil {
			return nil, err
		}
		value = result.Value()
	} else {
		value = dataMap
	}

	return json.Marshal(value)
}
