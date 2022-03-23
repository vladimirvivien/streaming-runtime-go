package k8s

import (
	"context"

	daprsubscriptions "github.com/dapr/dapr/pkg/apis/subscriptions/v2alpha1"
	streamingruntime "github.com/vladimirvivien/streaming-runtime/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	namespace string
	client.Client
}

func NewClient(namespace string) (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// setup scheme and client
	scheme := runtime.NewScheme()
	daprsubscriptions.AddToScheme(scheme)
	streamingruntime.AddToScheme(scheme)

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return &Client{
		namespace: namespace,
		Client:    c,
	}, nil
}

func (c *Client) GetSubscription(ctx context.Context, name string) (*daprsubscriptions.Subscription, error) {
	sub := daprsubscriptions.Subscription{}
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: c.namespace}, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

func (c *Client) GetStream(ctx context.Context, name string) (*streamingruntime.Stream, error) {
	stream := streamingruntime.Stream{}
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: c.namespace}, &stream); err != nil {
		return nil, err
	}
	return &stream, nil
}
