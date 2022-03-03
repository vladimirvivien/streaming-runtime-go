## Hello-Streaming Example

This directory contains a simple example that deploys a processor to processage streaming messages. The two components
of this example are:

* `message-gen` is a mock service that publishes message events to the defined topic
* `message-proc` is a processor that listens for topic events and processes them

### Running the example
Before you can run this example, you must have the following *pre-requisites*:

* Your cluster has the Dapr runtime components 
* Your cluster also needs to have the streaming-runtime components
* A streaming broker/provider (this example uses Redis Stream)

#### Install Dapr

This implementation of the Streaming-Runtime relies on Dapr. You must install the Dapr components on your cluster prior
to running the example.  

> See instructions on [installing Dapr components](https://docs.dapr.io/operations/hosting/kubernetes/kubernetes-deploy/)

#### Deploy a streaming broker

This example uses an event streaming platform to send messages from a producer to a message processor. You can use your
favorite broker including Kafka, Rabbit, Redis Streams, or any [pub/sub broker supported by Dapr](https://docs.dapr.io/reference/components-reference/supported-pubsub/)
for streaming. 

> This example comes with manifest file to set up Redis Streams as a pub/sub broker for the events.

#### Install the Streaming-Runtime controllers

You will need to install the Streaming-Runtime controller components before you can start.  This is done by simply 
running the following `kubectl` command:

```
kubectl apply -f https://github.com/vladimirvivien/streaming-runtime-go/blob/main/config/streaming-components.yaml
```


### Building components
These examples use `ko` for building and publishing container images as shown below.

#### Publishing `message-gen` image:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/hello-streaming/message-gen ko publish --bare ./message-gen
```

Publishing `message-proc` image:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/hello-streaming/message-proc ko publish --bare ./message-proc
```