# Processor component

A processor is user-provided code (deployed on the cluster) that can receive streamed events for processing. The code
can also send data downstream to another component or to a stream topic. Deploying a container image as a processor
automatically deploys the necessary bits to ensure the code receives data from its specified source.

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

The YAML above will deploy image `ghcr.io/vladimirvivien/streaming-runtime-examples/message-proc:latest` as a Processor
capable of receiving streamed data.  You can see an example of a processor [here](../examples/message-proc).