## Hello-Streaming Example

This directory contains a simple example that deploys a processor to processage streaming messages. The two components
of this example are:

* `message-gen` is a mock service that publishes message events to the defined topic
* `message-proc` is a processor that listens for topic events and processes them

### Building components
These examples use `ko` for building and publishing container images. These examples push to GitHub container repository.
However, you can push to wherever you host your container images.

#### Publishing `message-gen` image:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/hello-streaming/message-gen ko publish --bare ./message-gen
```

Publishing `message-proc` image:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/hello-streaming/message-proc ko publish --bare ./message-proc
```