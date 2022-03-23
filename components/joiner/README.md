# Joiner

The `Joiner` joins streaming elements from two `Stream` components within a specified time window.

## Joiner example

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Joiner
metadata:
  name: hello-goodbye-join
  namespace: default
spec:
  streams: # list of stream refs 
    - stream-name0
    - stream-name1
  window: "100ms"
  expression: "hello.salutation == goodbye.salutation"
  container:
    image:
  recipients:
    - messages
```

## Building components
This component ca be built and deployed using `ko` as is shown below.

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime/components/joiner ko publish --bare ./
```
