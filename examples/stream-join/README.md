# Stream-join example

This example demonstrates the use of the `Joiner` component to stream and join events from two separate streams of events.

## Components
This example uses several [streaming-runtime components](./manifests) as outlined below.

### Redis streaming
This example uses Redis streaming as pub/sub broker to stream and drain incoming events. See [redis.yaml](./manifests/redis.yaml).

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

### `ClusterStream`
A `ClusterStream` is component that configures the connection to the Redis streaming as a pub/sub broker. See [redis.yaml](./manifests/redis.yaml).

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

### The `Stream`s
A `Stream` is a component that defines (and creates, if possible) a stream topic. This example defines two separate
streams called `hello` and `goodbye` whose events will be joined using the Joiner component.
See [stream.yaml](./manifests/streams.yaml).

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Stream
metadata:
  name: hello
  namespace: default
spec:
  clusterStream: "redis-stream"
  topic: "hello"
  route: "/hello"

---
apiVersion: streaming.vivien.io/v1alpha1
kind: Stream
metadata:
  name: goodbye
  namespace: default
spec:
  clusterStream: "redis-stream"
  topic: "goodbye"
  route: "/goodbye"
```

### The `Joiner` component
The `Joiner` component, when configured, will stream events from both defined streams: `hello` and `goodbye`.
See [joiner.yaml](./manifests/joiner.yaml).

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Joiner
metadata:
  name: hello-goodbye-join
  namespace: default
spec:
  streams:
  - hello
  - goodbye
  window: 14s
  select:
    data: '{"hello": hello, "goodbye":goodbye}'
    where: "int(hello['id']) == 5 && goodbye.id == 5.0"
  servicePort: 8080
  target: message-proc/messages
```

### Message generator
Deployment `message-gen` deploys a simple [Go an application](../message-gen), that uses the Dapr API, to generate mock event messages
that are sent to each topic. See [message-gen.yaml](./manifests/message-gen.yaml).

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-message-gen
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        dapr.io/enabled: "true"
        dapr.io/app-id: "hello-message-gen"
    spec:
      containers:
        - name: message-gen
          image: ghcr.io/vladimirvivien/streaming-runtime-examples/message-gen:latest
          imagePullPolicy: Always
          env:
            - name: MESSAGE_EXPR # required: CEL expression for message
              value: '{"id": id, "greeting":"hello World!!", "timestamp":timestamp, "side":"hello"}'
            - name: CLUSTER_STREAM
              value: "redis-stream"
            - name: STREAM_TOPIC
              value: "hello"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: goodbye-message-gen
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        dapr.io/enabled: "true"
        dapr.io/app-id: "goodbye-message-gen"
    spec:
      containers:
        - name: message-gen
          image: ghcr.io/vladimirvivien/streaming-runtime-examples/message-gen:latest
          imagePullPolicy: Always
          env:
            - name: MESSAGE_EXPR # required: CEL expression for message
              value: '{"id": id, "greeting":"hello World!!", "timestamp":timestamp, "side":"goodbye"}'
            - name: CLUSTER_STREAM
              value: "redis-stream"
            - name: STREAM_TOPIC
              value: "goodbye"
```

### `Processor`
Component `message-proc` deploys a simple [Go application](../message-proc) that logs the  received event to standard output.
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

## Pre-requisites
Before you can run this example, you must have the following *pre-requisites*:

* Your cluster has the `dapr` runtime components deployed
* Your cluster also needs to have the `streaming-runtime` components
* A streaming broker/provider (this example uses Redis Stream)

## Installation
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

## Running the example
At this point, you are ready to run the example components.

### Deploy the components

The following command will deploy all of the components listed above, including the Redis streaming broker on the,
on the cluster:

```
kubectl apply -f https://github.com/vladimirvivien/streaming-runtime-go/blob/main/examples/stream-join/manifests
```

> NOTE: While this example uses Redis Streams, you can use any of your favorite brokers, including Kafka, Rabbit, NATS, etc., [supported by Dapr](https://docs.dapr.io/reference/components-reference/supported-pubsub/)
for streaming.

### Validate deployment
Validate that the expected components are deployed and are running OK.
First, get a list of running pods in the `default` namespace:

```
kubectl get pods
NAME                                   READY   STATUS    RESTARTS      AGE
goodbye-message-gen-744bc66fcc-79rvh   2/2     Running   1 (18s ago)   22s
hello-goodbye-join-7d57d8bf8b-mrh6k    2/2     Running   0             22s
hello-message-gen-66f9fb5bfc-n79x8     2/2     Running   1 (19s ago)   22s
message-proc-6799ffffc-27l4h           2/2     Running   0             22s
redis-6cc59df87c-qtfgr                 1/1     Running   0             22s
```

Next, retrieve the logs for pods `message-proc`

```
kubectl logs -l app=message-proc -c message-proc
2022/04/01 17:24:27 :8080 invoked: [content-type: application/json, url: ?, data: [{"goodbye":{"greeting":"hello World!!", "id":5, "side":"goodbye", "timestamp":"2022-04-01 17:24:15.521622316 +0000 UTC m=+28.031029898"}, "hello":{"greeting":"hello World!!", "id":5, "side":"hello", "timestamp":"2022-04-01 17:24:14.638525967 +0000 UTC m=+28.063013602"}}]
```