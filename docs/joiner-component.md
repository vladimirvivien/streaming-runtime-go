# Joiner

The `Joiner` component joins streaming elements from two `Stream` components within a specified time window.

* Originally, it will only support tumbling window
* Supports only JSON-encoded data
* Support for data selection and data filtering expressions

## Joiner example

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Joiner
metadata:
  name: hello-goodbye-join
  namespace: default
spec:
  streams: # list of stream referenced
    - hello
    - goodbye
  window: 14s
  select: # event filter and selection expressions using CEL
    data: "string(hello.message.id) + '~' + hello.message.text"
    where: "int(hello.message['id']) > 5 && goodbye.message.id >= 5.0"
  servicePort: 8080

  # target: the component[/route] where to send joined messages
  # if route is not provided, component/component is used for routing.
  target: message-proc/messages

  # Optional spec.container section to specify image of Joiner component
  # If not provided, the latest version will be used
  container:
    name: message-proc
    image: ghcr.io/vladimirvivien/streaming-runtime/components/joiner:latest
    imagePullPolicy: Always
```

In the previous yaml snippet, joiner `hello-goodbye-join` is set up to receive data from upstream topics
`hello` and `goodbye` within a 14 sec window. The component uses `spec.select` to specify data selection and
data filtering expressions:

```yaml
  select: # event filter and selection expressions using CEL
    data: "string(hello.message.id) + '~' + hello.message.text"
    where: "int(hello.message['id']) > 5 && goodbye.message.id >= 5.0"
```

Element `spec.select.data` specifies an expression for specifying how to shape the data collected. Element
`spec.select.where` specifies an expression to specifying how to filter the streaming events.

> See the full example for joiner [here](../examples/stream-join).