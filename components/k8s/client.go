package k8s

import (
	"context"

	daprcomponents "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	daprsubscriptions "github.com/dapr/dapr/pkg/apis/subscriptions/v2alpha1"
	componentsK8s "github.com/dapr/dapr/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var subscriptionGVR = schema.GroupVersionResource{Group: "dapr.io", Version: "v2alpha1", Resource: "subscriptions"}

type Client struct {
	namespace        string
	componentsClient componentsK8s.Interface
	dynaClient       dynamic.Interface
}

func NewClient(namespace string) (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	compClient, err := componentsK8s.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynaC, err := dynamic.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	return &Client{componentsClient: compClient, dynaClient: dynaC, namespace: namespace}, nil
}

func (c *Client) GetComponent(name string) (*daprcomponents.Component, error) {
	component, err := c.componentsClient.ComponentsV1alpha1().Components(c.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return component, nil
}

func (c *Client) GetSubscription(ctx context.Context, name string) (*daprsubscriptions.Subscription, error) {
	unstructuredSub, err := c.dynaClient.Resource(subscriptionGVR).Namespace(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	sub := new(daprsubscriptions.Subscription)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredSub.UnstructuredContent(), sub); err != nil {
		return nil, err
	}
	return sub, nil
}
