# Channel

A `Channel` is a component that can be used to route data from one component to another with support for
the ability to filter and shape the output data.

> Read more about `Joiner` [here](../../docs/channel-component.md).

## Building components
This component ca be built and deployed using `ko` as is shown below.

```
KO_DOCKER_REPO=ghcr.io/vladimirvivien/streaming-runtime/components/channel ko publish --bare ./
```
