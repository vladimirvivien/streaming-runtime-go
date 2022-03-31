# message-gen

A component used to generate and send fake messages to a specified pub/sub topic used for testing.

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
          - name: MESSAGE_EXPR # required: CEL expression for message
            value: ''{"id": id, "greeting":"hello", "location":"world", "timestamp":timestamp}''
          - name: CLUSTER_STREAM
            value: "redis-stream"
          - name: STREAM_TOPIC
            value: "hello-topic-stream"
```
The followings are pre-generated dynamic values available to be used in the message expression:
* `id` an incremental counter value 
* `timestamp` a time value generated for each message sent

## Build and publish

This component can be built with `ko`, similarly as shown below:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/message-gen ko publish --bare ./
```
