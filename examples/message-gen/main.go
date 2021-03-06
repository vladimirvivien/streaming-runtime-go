package main

import (
	"context"
	"log"
	"math"
	"os"
	"reflect"
	"strconv"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	commontypes "github.com/google/cel-go/common/types/ref"
	exprv1alpha1 "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	messageCountEnv = os.Getenv("MESSAGE_COUNT")
	pubsubName      = os.Getenv("CLUSTER_STREAM")
	topicName       = os.Getenv("STREAM_TOPIC")
	messageExpr     = os.Getenv("MESSAGE_EXPR")
	messageDelayEnv = os.Getenv("MESSAGE_DELAY")

	messageCount = 0
)

func main() {
	client, err := dapr.NewClient()
	if err != nil {
		log.Fatalf("message-gen: failed to create client: %s", err)
	}

	if messageCountEnv != "" {
		if count, err := strconv.Atoi(messageCountEnv); err == nil{
			messageCount = count
		}
	}

	if pubsubName == "" {
		log.Fatal("pubsub name must be provided")
	}

	if topicName == "" {
		log.Fatal("topic name must be provided")
	}

	if messageExpr == "" {
		log.Fatal("message expression must be provided")
	}

	if messageDelayEnv == "" {
		messageDelayEnv = "1s"
	}
	delay, err := time.ParseDuration(messageDelayEnv)
	if err != nil {
		log.Printf("message-gen: invalid delay value: %s, default to 1s", err)
		delay = time.Second * 3
	}

	log.Printf("message-gen: client created: clusterStream: %s, topic: %s, message-expr: %s", pubsubName, topicName, messageExpr)

	prog, err := compileCELProg(messageExpr, map[string]*exprv1alpha1.Type{"timestamp": decls.String, "id": decls.Int})
	if err != nil {
		log.Fatalf("message-gen: failed to compile expression: %s", err)
	}

	defer client.Close()
	ctx := context.Background()

	if messageCount == 0 {
		messageCount = math.MaxInt
	}

	for n := 0; n <= messageCount; n++ {
		val, _, err := prog.Eval(map[string]interface{}{
			"timestamp": time.Now().String(),
			"id":   n + 1,
		})
		if err != nil {
			log.Fatalf("messasge-gen: fail to evaluate message expression: %s", err)
		}
		switch val.Type().TypeName(){
		default:
			json, err := marshalJSON(val)
			if err != nil {
				log.Fatalf("message-gen: fail to convert to JSON: %s", err)
			}
			if err := client.PublishEvent(ctx, pubsubName, topicName, json, dapr.PublishEventWithContentType("application/json")); err != nil {
				log.Fatalf("message-gen: fail to publish event: %s", err)
			}
			log.Printf("message-gen: message sent: %s", string(json))
			time.Sleep(delay)
		case "list":
			list, ok := val.Value().([]commontypes.Val)
			if !ok {
				log.Fatalf("message-gen: unexpected list type: %T", val.Value())
			}
			for _, l := range list {
				json, err := marshalJSON(l)
				if err != nil {
					log.Fatalf("message-gen: fail to convert to JSON: %s", err)
				}
				if err := client.PublishEvent(ctx, pubsubName, topicName, json, dapr.PublishEventWithContentType("application/json")); err != nil {
					log.Fatalf("message-gen: fail to publish event: %s", err)
				}
				log.Printf("message-gen: message sent: %s", string(json))
				time.Sleep(delay)
				n++
			}
		}
	}
	log.Printf("message-gen: message count reached: %d",messageCount)
	select {} // stay alive to avoid pod restart
}

func compileCELProg(expr string, variables map[string]*exprv1alpha1.Type) (cel.Program, error) {
	var varDecls []*exprv1alpha1.Decl
	for variable, vartype := range variables {
		varDecls = append(varDecls, decls.NewVar(variable, vartype))
	}
	d := cel.Declarations(varDecls...)
	env, err := cel.NewEnv(d)
	if err != nil {
		return nil, err
	}

	// compile and check for errs
	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	prog, err := env.Program(ast)
	if err != nil {
		return nil, err
	}
	return prog, nil
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
