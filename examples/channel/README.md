# Channel example

This example demonstrates the use of the `Channel` component:
* Allows data to be streamed from pubsub topic or another component
* Supports data filter and data selection expressions
* Able to send data to another stream or another component

## Components
This example uses several [streaming-runtime components](./manifests) as shown in the illustration below.

![Components](channel-example.png "Components")


## Pre-requisites

Before you can run this example, you must have the following *pre-requisites*:

* Your cluster has the `dapr` runtime components deployed
* Your cluster also needs to have the `streaming-runtime-go` components deployed
* A Redis Streams and RabbitMQ brokers

### Pre-install RabbitMQ

This example uses RabbitMQ to receive processed streamed messages. Use the Helm chart for RabbitMQ for a quick deployment - https://github.com/bitnami/charts/tree/master/bitnami/rabbitmq.
Follow instruction, from Helm chart installation output, to get the namespace, username, password, and the server port.
Then, configure the ClusterStream accordingly. Set the `host` to match the name of the Rabbit Kubernetes service and
its namespace (i.e. `<rabbitmq-service-name>.<namespace>.svc.cluster.local:<port>`)

## Install and run

For this simple example, the following steps will install the components necessary to generate and stream events using
Redis that are then processed by a simple component.

### Install Dapr

This implementation of the Streaming-Runtime uses on Dapr and its API. You must install the Dapr components on your cluster prior
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

#### Deploy the components

The following command will deploy all components to run the example on the cluster:

```
kubectl apply -f https://github.com/vladimirvivien/streaming-runtime-go/blob/main/examples/channel/manifests
```

> NOTE: While this example uses Redis Streams and RabbitMQ, you can use any of your favorite brokers including Kafka, NATS, etc., [supported by Dapr](https://docs.dapr.io/reference/components-reference/supported-pubsub/)
for streaming.

#### Validate deployment

Validate that the expected components are deployed and are running OK.
First, get a list of running pods in the `default` namespace:

```
kubectl get pods
NAME                                 READY   STATUS    RESTARTS       AGE
greetings-channel-5898785869-6zp4x   2/2     Running   34 (74m ago)   121m
message-gen-549b5db8ff-5phwr         2/2     Running   5 (120m ago)   121m
message-proc-686dcf6d58-gwnvk        2/2     Running   2 (120m ago)   121m
rabbitmq-0                           1/1     Running   0              4h16m
redis-6cc59df87c-wf9sv               1/1     Running   0              121m
```

If everything is working OK, you should be able to see all messages sent to the RabbitMQ queue forwarded to the message-proc component:

```
kubectl logs -l app=message-proc -c message-proc
2022/04/05 17:35:49 :8080 invoked: [content-type: application/cloudevents+json, url: ?, data: {"datacontenttype":"application/json","traceid":"00-0afc6c7fc14595e5ffcfce367262492b-7c3b52a67a2f3ad8-00","tracestate":"","data":{"newgreeting":"hello world!"},"id":"e4c3ea8e-f0d0-48f6-8b98-44f0c555bbd6","specversion":"1.0","source":"greetings-channel","type":"com.dapr.event.sent","topic":"greetings-sink","pubsubname":"rabbit-stream"}
2022/04/05 17:36:04 :8080 invoked: [content-type: application/cloudevents+json, url: ?, data: {"specversion":"1.0","topic":"greetings-sink","traceid":"00-857cb6f6cebf144964f16e8e5e506c15-4afa2cd2ab1f21e9-00","type":"com.dapr.event.sent","pubsubname":"rabbit-stream","tracestate":"","data":{"newgreeting":"hello world!"},"id":"710f0cfa-ba84-41a2-a899-497c9e6a28d1","datacontenttype":"application/json","source":"greetings-channel"}
```

## Manifest artifacts

### Redis Streams deployment
This example uses Redis Stream from which events are streamed before they are processed. See [redis.yaml](./manifests/redis.yaml).

> The following YAML deploys a single-pod instance of Redis stream for simplicity. You can use an operator or
> a Helm chart for a more sophisticated installation.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: default
spec:
  selector:
    matchLabels:
      app: redis
  replicas: 1
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:6.2.6-alpine
```

### Redis `ClusterStream`
This `ClusterStream` component configures the connection to Redis Stream as a pub/sub broker. See [redis.yaml](./manifests/redis.yaml).

> Note that this component expects the broker to be deployed and accessible ahead of time.

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

### Redis  `Stream`
This `Stream` component defines (and creates, if possible) a stream topic where messages will be streamed from. 
See [stream.yaml](./manifests/stream.yaml).

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Stream
metadata:
  name: greetings
  namespace: default
spec:
  clusterStream: redis-stream
  topic: greetings
  route: /greetings
  recipients:
    - greetings-channel
```

> Note that the `greetings` Stream component targets the `greetings-channel` Channel component as its recipient (see further below).

### RabbitMQ `ClusterStream`
This `ClusterStream` component configures a connection to the RabbitMQ broker. See [redis.yaml](./manifests/rabbit.yaml).

> Note that this component expects the broker to be already be deployed and accessible ahead of time.

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: ClusterStream
metadata:
  name: rabbit-stream
  namespace: default
spec:
  protocol: rabbitmq
  properties:
    host: amqp://user:PfobN4Ttfq@rabbitmq.default.svc.cluster.local:5672
```
The connection string for `host` uses format `amqp://<username>:<password>@<rabbitserver>:<port>`. If you use the steps,
in the pre-requisites further below, to install Rabbit (with Helm), you will find that information from the Helm output.

### RabbitMQ  `Stream`
This `Stream` component defines (and creates, if possible) a RabbitMQ topic (queue) where processed messages will be sent.

See [stream.yaml](./manifests/stream.yaml).

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Stream
metadata:
  name: greetings-sink
  namespace: default
spec:
  clusterStream: rabbit-stream
  topic: greetings-sink
  route: /greetings-sink
  recipients:
    - message-proc
```

### The `Channel` component

The `Channel` connects the streamed events from Redis and routes them to the RabbitMQ queue:
* It streams data from the Stream component `redis-stream/greetings` topic.
* Applies data filtering and data selection expressions
* Then forwards the newly created data objects to RabbitMQ stream `rabbit-strea/greetings-sink`

See [channel.yaml](./manifests/channel.yaml).

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Channel
metadata:
  name: greetings-channel
  namespace: default
spec:
  servicePort: 8080
  serviceRoute: greetings
  select:
    data: '{"new-greeting": greetings.greeting + " " + greetings.location + "!"}'
    where: "int(greetings['id']) % 5 == 0"
  target:
    stream: rabbit-stream/greetings-sink
```

### The message `Processor`
Component `message-proc` deploys a simple [Go application](../message-proc) that logs (standard output) messages that
are sent to the RabbitMQ queue.

See [message-proc](./manifests/message-proc.yaml).

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

### Message generator
Lastly, the `message-gen` application deploys a simple [Go an application](../message-gen) that generates mock event messages
that are sent to Redis stream. See [message-gen.yaml](./manifests/message-gen.yaml).

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: message-gen
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        dapr.io/enabled: "true"
        dapr.io/app-id: "message-gen"
    spec:
      containers:
        - name: message-gen
          image: ghcr.io/vladimirvivien/streaming-runtime-examples/message-gen:latest
          imagePullPolicy: Always
          env:
            - name: MESSAGE_EXPR # required: CEL expression for message
              value: '{"id": id, "greeting":"hello", "location":"world", "timestamp":timestamp}'
            - name: CLUSTER_STREAM
              value: "redis-stream"
            - name: STREAM_TOPIC
              value: "greetings"
```

