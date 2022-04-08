# Monitor datacenter power usage

This example demonstrates the use of the `Joiner` component to stream (mocked), join, and filter power usage events from
compute equipments from a datacenter. This example is inspired by [datacenter power usage](https://developer.confluent.io/tutorials/datacenter/confluent.html) example from
Confluent (however, for simplicity's sake, it streams from Redis and RabbitMQ instead of Kafka).

## Components
This example uses several [streaming-runtime components](./manifests) as shown in the illustration below.

![Components](poweruse-components.png "Components")

## Pre-requisites

Before you can run this example, you must have the following *pre-requisites*:

* Your cluster has the `dapr` runtime components deployed
* Your cluster also needs to have the `streaming-runtime-go` components deployed
* Deploy a Redis cluster (this example deploys a one-node cluster for this demonstration)

## Install and run

For this simple example, the following steps will install the components necessary.

### Install Dapr

This implementation of the Streaming-Runtime project uses Dapr and its API. You must install the Dapr components on your cluster prior
to running the example. Install the [Dapr CLI](https://docs.dapr.io/getting-started/install-dapr-cli/) and run the following
command to install the Dapr components on the Kubernetes cluster

```
dapr init -k
```

> See instructions on [installing Dapr components](https://docs.dapr.io/operations/hosting/kubernetes/kubernetes-deploy/)

### Install the Streaming-Runtime controllers

You will need to install the Streaming-Runtime controller components before you can start.  This is done by simply
running the following `kubectl` command:

```
kubectl apply -f https://raw.githubusercontent.com/vladimirvivien/streaming-runtime-go/main/config/streaming-components.yaml
```

### Running the example

At this point, you are ready to run the example components.

#### Deploy the components

The following command will deploy all components to run the example on the cluster:

```
kubectl apply -f https://raw.githubusercontent.com/vladimirvivien/streaming-runtime-go/main/examples/power-usage/manifests-all.yaml
```

>NOTE: if you use a different username/password for RabbitMQ, download the manifest file above first. Then, update 
> the username and password for the `rabbit-stream`  ClusterStream component.

#### Validate deployment

Validate that the expected components are deployed and are running OK.
First, get a list of running pods in the `default` namespace:

```
kubectl get pods
NAME                                     READY   STATUS    RESTARTS      AGE
billable-proc-6bb87df578-nr5jj           2/2     Running   0             37m
join-tenants-poweruse-7b795c9d8b-wlbrn   2/2     Running   1 (37m ago)   37m
poweruse-gen-74f8fd884c-m7vfq            2/2     Running   1 (37m ago)   37m
rabbitmq-0                               1/1     Running   0             77m
redis-6cc59df87c-m52h2                   1/1     Running   0             43m
tenants-gen-5f44dbcb8f-h4x6h             2/2     Running   1 (37m ago)   37m
```

If everything is working OK, you will see billable tenants information streaming to the `billable-proc` component with
the following command:

```
kubectl logs -l app=billable-proc -c billable-proc
2022/04/07 16:24:01 Data received: [{"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}]
2022/04/07 16:24:21 Data received: [{"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}]```
2022/04/07 16:24:41 Data received: [{"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1034}, {"customer_id":924, "tenant_id":"dc:eqix:us:chi1:12", "tenant_kwh_usage":1134}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1201}, {"customer_id":243, "tenant_id":"dc:kddi:cn:hnk2:11", "tenant_kwh_usage":1119}]
```
## Artifacts manifest

### Redis Streams deployment
This example uses Redis Streams to receive and stream system log events before they are processed. See [redis.yaml](./manifests/redis.yaml).

> The following YAML deploys a single-pod instance of Redis stream for simplicity. You can use an operator or
> a Helm chart for a more sophisticated installation.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: default
spec:
  selector:
    matchLabels:
      app: redis
  replicas: 1
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:6.2.6-alpine
```

### Redis `ClusterStream`
This `ClusterStream` component configures the connection to Redis Streams as a pub/sub broker. See [redis.yaml](./manifests/redis.yaml).

> Note that this component expects the broker to be deployed and accessible ahead of time.

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: ClusterStream
metadata:
  name: syslog-stream
  namespace: default
spec:
  protocol: redis
  properties:
    redisHost: redis:6379
    redisPassword: ""
```

### Redis `Stream`
This `Stream` component defines (and creates, if possible) a stream topic where syslog events will be streamed from.
See [stream.yaml](./manifests/stream.yaml).

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Stream
metadata:
  name: greetings
  namespace: default
spec:
  clusterStream: "syslog-stream"
  topic: "syslog"
  route: "/syslog"
  recipients:
    - syslog-channel
```

> Note that the `syslog` Stream component targets the `syslog-channel` Channel component as its recipient (see below).

### RabbitMQ `ClusterStream`
This `ClusterStream` component configures a connection to the RabbitMQ broker. See [rabbit.yaml](./manifests/rabbit.yaml).

> Note that this component expects the broker to already be deployed and accessible ahead of time.

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: ClusterStream
metadata:
  name: rabbit-stream
  namespace: default
spec:
  protocol: rabbitmq
  properties:
    host: amqp://user:PfobN4Ttfq@rabbitmq.default.svc.cluster.local:5672
```

### The `Joiner` component
The `Joiner` component, when configured, will stream events from both defined streams: `tenants` and `poweruse`.
See [joiner.yaml](./manifests/join.yaml).

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Joiner
metadata:
  name: join-tenants-poweruse
  namespace: default
spec:
  servicePort: 8080
  window: 20s
  stream:
    from:
      - tenants
      - poweruse
    to:
      - component: billable-proc/billables
    select: |
      {
        "customer_id": tenants.customer_id,
        "tenant_id": tenants.tenant_id,
        "tenant_kwh_usage": poweruse.kwh_use
      }
    where: |
      tenants.tenant_id == poweruse.tenant_id
        && poweruse.power_use >= 1.0
```

### The message `Processor`
Component `billable-proc` deploys a simple [Go application](../message-proc) that logs (to standard output) the billable
tenant events that are clculated and streamed from the joined streams.

See [message-proc](./manifests/billable-proc.yaml).

```yaml
apiVersion: streaming.vivien.io/v1alpha1
kind: Processor
metadata:
  name: syslog-proc
  namespace: default
spec:
  replicas: 1
  servicePort: 8080
  serviceRoute: syslog
  container:
    name: syslog-proc
    image: ghcr.io/vladimirvivien/streaming-runtime-examples/message-proc:latest
    imagePullPolicy: Always
```

### Tenant and power use data generator
Lastly, the `tenant-gen` and `poweruse-gen` applications are a simple [Go an application](../message-gen) that generates 
mock datacenter tenant data that are sent to both the Redis Streams and RabbitMQ topics/queue respectively. 

See [tenants-gen.yaml](./manifests/tenants-gen.yaml) and [poweruse-gen.yaml](./manifests/poweruse-gen.yaml).

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: poweruse-gen
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        dapr.io/app-id: poweruse-gen
        dapr.io/enabled: "true"
    spec:
      containers:
        - env:
            - name: MESSAGE_EXPR
              value: |
                [
                  {"panel_id": 1, "tenant_id": "dc:eqix:us:chi1:12", "power_use": 1.05, "kwh_use": 1034},
                  {"panel_id": 2, "tenant_id": "dc:eqix:us:chi1:10", "power_use": 0.85, "kwh_use": 867},
                  {"panel_id": 1, "tenant_id": "dc:kddi:eu:ber1:15", "power_use": 0.54, "kwh_use": 345},
                  {"panel_id": 2, "tenant_id": "dc:kddi:eu:ber1:20", "power_use": 0.67, "kwh_use": 288},
                  {"panel_id": 1, "tenant_id": "dc:kddi:cn:hnk2:11", "power_use": 1.11, "kwh_use": 1119},
                  {"panel_id": 1, "tenant_id": "dc:eqix:us:chi1:12", "power_use": 1.01, "kwh_use": 1134},
                  {"panel_id": 2, "tenant_id": "dc:eqix:us:chi1:10", "power_use": 0.75, "kwh_use": 898},
                  {"panel_id": 1, "tenant_id": "dc:kddi:cn:hnk2:11", "power_use": 1.10, "kwh_use": 1201},
                ]
            - name: CLUSTER_STREAM
              value: rabbit-stream
            - name: STREAM_TOPIC
              value: poweruse
          image: ghcr.io/vladimirvivien/streaming-runtime-examples/message-gen:latest
          imagePullPolicy: Always
          name: poweruse-gen
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tenants-gen
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        dapr.io/app-id: tenants-gen
        dapr.io/enabled: "true"
    spec:
      containers:
        - env:
            - name: MESSAGE_EXPR
              value: |
                [
                  {"tenant_id":"dc:eqix:us:chi1:12", "customer_id": 924},
                  {"tenant_id":"dc:eqix:us:chi1:10", "customer_id": 243},
                  {"tenant_id":"dc:kddi:eu:ber1:15", "customer_id": 924},
                  {"tenant_id":"dc:kddi:eu:ber1:20", "customer_id": 123},
                  {"tenant_id":"dc:kddi:cn:hnk2:11", "customer_id": 243},
                ]
            - name: CLUSTER_STREAM
              value: redis-stream
            - name: STREAM_TOPIC
              value: tenants
          image: ghcr.io/vladimirvivien/streaming-runtime-examples/message-gen:latest
          imagePullPolicy: Always
          name: tenants-gen
```


