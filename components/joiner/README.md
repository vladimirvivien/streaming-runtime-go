# Joiner

The `Joiner` joins streaming elements from two `Stream` components within a specified time window. The component also
supports the ability to specify data filtering and selection expressions for the aggregated elements.

> Read more about `Joiner` [here](../../docs/joiner-component.md).

## Building components
This component ca be built and deployed using `ko` as is shown below.

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime/components/joiner ko publish --bare ./
```
