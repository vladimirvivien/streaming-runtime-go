## Hello-Streaming Example

This directory contains a simple example that deploys a Processor component to process streaming event data. 

### Components
The example uses several [streaming-runtime components](./manifests/hello-streaming.yaml) to define an event stream and a processor service that will receive the messages
that will receive the events for processing.

#### `ClusterStream`
Defines the connection to a broker. The example below defines a ClusterStream named `redis-stream` that uses a Redis Streams event broker.

> Note that this component expects the broker to be already deployed and accessible ahead of time.

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: ClusterStream
metadata:
  name: redis-stream
  namespace: default
spec:
  protocol: redis
  properties:
    redisHost: redis:6379
    redisPassword: ""
```

#### `Stream`

Stream is a component that defines (and creates, if possible) a stream topic. The stream also specifies a recipient component
that will receive the messages.

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Stream
metadata:
  name: hello-topic-stream
  namespace: default
spec:
  clusterStream: "redis-stream"
  topic: "hello-topic-stream"
  route: "/messages"
  recipients:
    - message-proc
```

#### `Processor`
The processor is an application that can stream events from a components that can emit events such as Streams. The
processor advertises an HTTP/gRPC port where it can receive data as shown below

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Processor
metadata:
  name: message-proc
  namespace: default
spec:
  replicas: 1
  servicePort: 8080
  container:
    name: message-proc
    image: ghcr.io/vladimirvivien/streaming-runtime-examples/hello-streaming/message-proc:latest
    imagePullPolicy: Always
```
### Pre-requisites
Before you can run this example, you must have the following *pre-requisites*:

* Your cluster has the Dapr runtime components 
* Your cluster also needs to have the streaming-runtime components
* A streaming broker/provider (this example uses Redis Stream)

### Install Dapr

This implementation of the Streaming-Runtime relies on Dapr. You must install the Dapr components on your cluster prior
to running the example.  

> See instructions on [installing Dapr components](https://docs.dapr.io/operations/hosting/kubernetes/kubernetes-deploy/)

### Install the Streaming-Runtime controllers

You will need to install the Streaming-Runtime controller components before you can start.  This is done by simply 
running the following `kubectl` command:

```
kubectl apply -f https://github.com/vladimirvivien/streaming-runtime-go/blob/main/config/streaming-components.yaml
```

### Running the example
At this point, you are ready to run the example components. 

#### Event streaming broker

This example uses an event streaming broker to send messages from a message generator (message-gen) to a message processor (message-proc).
Use the following to deploy Redis (ensure to use latest versions with streams support):

```
kubectl apply -f https://github.com/vladimirvivien/streaming-runtime-go/blob/main/examples/hello-streaming/manifests/redis.yaml
```

While this example uses Redis Streams, you can use any of your favorite brokers, including Kafka, Rabbit, NATS, etc., [supported by Dapr](https://docs.dapr.io/reference/components-reference/supported-pubsub/)
for streaming.

#### Deploy application
The application can be deployed by applying the following `kubectl` command:

```
kubectl apply -f https://github.com/vladimirvivien/streaming-runtime-go/blob/main/examples/hello-streaming/manifests/hello-streaming.yaml
```

The previous command will deploy two applications:

* [`message-gen`](../message-gen) is a simple message generator that sends events to a defined topic.
* [`message-proc`](./message-proc) is a processor that will receive events from the topic for processing 

In this simple example, the processor simply logs the incoming message. You can validate that it is running OK by
printing out the log statements from the running container. First, get a list of running pods in the `default` namespace:

```
kubectl get pods
NAME                            READY   STATUS    RESTARTS      AGE
message-gen-696747d4f4-kzxsc    2/2     Running   1 (48m ago)   48m
message-proc-796f5db478-tk9np   2/2     Running   0             48m
redis-6cc59df87c-kw6ll          1/1     Running   0             48m
```

Next, retrieve the logs for pods `message-proc`

```
kubectl logs -l app=message-proc -c message-proc
2022/03/03 17:22:33 /messages invoked: [content-type: application/cloudevents+json, url: ?, data: {"traceid":"00-68a56b48c90ba65167036da7ab0f6a9d-4dc61f63ce5fbea5-00","data":{"message":{"id":440,"text":"time is 2022-03-03 17:22:33.053691534 +0000 UTC m=+3075.855265688"}},"specversion":"1.0","datacontenttype":"application/json","source":"message-gen","pubsubname":"redis-stream","tracestate":"","id":"04a5586b-72de-4c3c-ba2f-922e1147a00f","type":"com.dapr.event.sent","topic":"hello-topic-stream"}
```
### Building components
This example uses `ko` for building and publishing the processor container image as shown below.

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/hello-streaming/message-proc ko publish --bare ./message-proc
```