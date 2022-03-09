# message-gen

This component can be used to generate and send messages to dapr-specified pub/sub topic.

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
          image: ghcr.io/vladimirvivien/streaming-runtime-examples/hello-streaming/message-gen:latest
          imagePullPolicy: Always
          env:
          - name: CLUSTER_STREAM
            value: "redis-stream"
          - name: STREAM_TOPIC
            value: "hello-topic-stream"
```

## Build and publish

This component can be built with `ko`, similarly as shown below:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/message-gen ko publish --bare ./
```
