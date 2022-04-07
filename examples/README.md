# Streaming runtime examples

This directory contains several examples that are designed to showcase its components.

* [Hello-streaming](./hello-streaming) - Get started here with this simple example with instructions on setting your environment for this and other examples.
* [Channel](./channel) - This example showcases the [Channel](../docs/channel-component.md) that uses common-language expressions to filter  and stream eventes from a stream source.
* [SSH attack detection](./detect-ssh-attack) - This example uses the [Channel](../docs/channel-component.md) detect invalid SSH logins from streaming syslog events.
* [Stream Join](./stream-join) - This example showcases the [Joiner](../docs/joiner-component.md) that uses common-language expressions to filter, stream, and join events from different stream sources.
* [Datacenter power usage](./power-usage) - This examples uses the [Joiner](../docs/joiner-component.md) to stream data from a Redis and a RabbitMQ streams to analyze power usage from a fictitious datacenter.
* [Message processor](./message-proc) - This example is a simple Go application that shows how to create an event processor applications that can receive streamed data from other components.