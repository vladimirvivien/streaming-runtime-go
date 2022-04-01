# message-proc

This code uses the Dapr API to receive incoming streaming events from a configured streaming topic.  The code can be
deployed as a `Processor` component which automatically creates the necessary deployment for getting the pod running
on the cluster.

## Usage

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Processor
metadata:
  name: message-proc
  namespace: default
spec:
  replicas: 1
  servicePort: 8080
  serviceRoute: messages
  container:
    name: message-proc
    image: ghcr.io/vladimirvivien/streaming-runtime-examples/message-proc:latest
    imagePullPolicy: Always
```
The previous YAML deploys the code as a processor component and will listen for
incoming event route `/messages`.  For a full example of how message-proc component is used,
see the [hello-streaming](../hello-streaming) example.

## Build and publish

This component can be built with `ko`, similarly as shown below:

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime-examples/message-proc ko publish --bare ./
```
