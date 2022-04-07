# Channel component

A `Channel` is a component that can be used to route data from one component to another with support for
the data filtration and assembly expressions (using the common language expression) to filter and construct the output data.

## Channel example

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Channel
metadata:
  name: greetings-channel
  namespace: default
spec:
  servicePort: 8080
  serviceRoute: greetings
  
  select: # data filter and construction expressions using CEL
    data: '{"newgreeting": greetings.greeting + " " + greetings.location + "!"}'
    where: "int(greetings['id']) % 5 == 0"
    
  # target: specifies the target where streamed output will be sent
  # Data can be sent to another component or to a pubsub stream    
  target:
    stream: rabbit-stream/greetings-sink

  # Optional spec.container section to specify image of Channel component
  # If not provided, the latest version will be used
  container:
    name: message-proc
    image: ghcr.io/vladimirvivien/streaming-runtime/components/channel:latest
    imagePullPolicy: Always
```

In the previous yaml snippet, channel `greeting-channel` is set up to receive streamed events while applying
the following data filtration and assembly expressions:

```yaml
  select: 
    data: '{"newgreeting": greetings.greeting + " " + greetings.location + "!"}'
    where: "int(greetings['id']) % 5 == 0"

```

> See the full example  [here](../examples/channel).