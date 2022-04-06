# message-gen

This code uses the Dapr API to connect to (an already established pub/sub) topic and publish a stream of generated
events.  The code uses `MESSAGE_EXPR` to define a JSON value containing generated message.

## Usage

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: message-gen
  labels:
    app: message-gen
spec:
  replicas: 1
  selector:
    matchLabels:
      app: message-gen
  template:
    metadata:
      labels:
        app: message-gen
      annotations:
        dapr.io/enabled: "true"
        dapr.io/app-id: "message-gen"
    spec:
      containers:
        - name: message-gen
          image: ghcr.io/vladimirvivien/streaming-runtime-examples/message-gen:latest
          imagePullPolicy: Always
          env:
          - name: MESSAGE_COUNT # optional count value
            value: "200"
          - name: MESSAGE_DELAY # optional: delay between each message sent
            value: "1s"
          - name: MESSAGE_EXPR # required: CEL expression for message
            value: ''{"id": id, "greeting":"hello", "location":"world", "timestamp":timestamp}''
          - name: CLUSTER_STREAM
            value: "redis-stream"
          - name: STREAM_TOPIC
            value: "hello-topic-stream"
```
The source code also provides pre-generated dynamic values that are made available to be injected in the message expression:
* `id` an incremental counter value 
* `timestamp` a time value generated for each message sent

## Build and publish

This component can be built with `ko`, similarly as shown below:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/message-gen ko publish --bare ./
```
