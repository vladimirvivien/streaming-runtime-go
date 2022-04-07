# message-proc

This code uses the Dapr API to receive incoming streaming events from a configured streaming topic.  The code can be
deployed as a [Processor](../../docs/processor-component.md) component which automatically creates the necessary deployment for getting the pod running
on the cluster.

Read more about the Processor [here](../../docs/processor-component.md).

## Build and publish

This component can be built with `ko`, similarly as shown below:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/message-proc ko publish --bare ./
```
