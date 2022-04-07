# message-proc

This simple Go program is designed to be deployed as a [Processor](../../docs/processor-component.md) component that
receives streaming events from a specified topic.  When deployed as a processor, it automatically creates the necessary 
deployment for getting the pod running on the cluster.

Read more about the Processor component [here](../../docs/processor-component.md).

## Build and publish

This component can be built with `ko`, similarly as shown below:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/message-proc ko publish --bare ./
```
