# Stream-join example

This example shows how to use the Joiner component to join event data from two topic sources.

## Components
* Single-node Redis-streams deployment
* ClusterStream for the Redis stream
* Two `Stream`s components: `hello` and `goodbye`
* A `Joiner` component that streams events from streams `hello` and `goodbye`.
  * The joiner component also applies filtering and data selection expressions
* `message-gen` is a simple Go program that emits events on the streams
* `message-proc` is a simple `Processor` that is used to receive the output of the Joiner