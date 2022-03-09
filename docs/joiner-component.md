# Joiner

The `Joiner` joins streaming elements from two `Stream` components within a specified time window.

* Originally, it will only support tumbling window
* Supports only left-join semantics when applying join expressions
* Supports only JSON-encoded date
* Ability to specify an expression used to join the data

## Joiner example

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Joiner
metadata:
  name: hello-goodbye-join
  namespace: default
spec:
  leftPath: "hello"
  rightPath: "goodbye"
  window: "100ms"
  expression: "hello.salutation == goodbye.salutation"
  container:
    image:
  recipients:
    - messages
```

In the previous yaml snippet, joiner `hello-goodbye-join` is set up to receive data from upstream topics on paths
`hello` and `goodbye` within a 100 milliseconds window where `hello.salutation == goodbye.salutation` is true. Joined
data will be forwarded to recipient component `messages`.