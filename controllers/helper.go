package controllers

import (
	"encoding/json"
	"fmt"
	"log"

	daprcomponents "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	daprsubscriptions "github.com/dapr/dapr/pkg/apis/subscriptions/v2alpha1"
	streamingruntime "github.com/vladimirvivien/streaming-runtime/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getMetadataStringValues(keyName string, properties map[string]string) (result []string) {
	for key, value := range properties {
		if key == keyName {
			result = append(result, value)
		}
	}
	return
}

// createDaprComponent is a helper method to create v1alpha1 Dapr Component API objects
func (r *ClusterStreamReconciler) createDaprComponent(componentType string, clusterStream *streamingruntime.ClusterStream) (*daprcomponents.Component, error) {
	properties := clusterStream.Spec.Properties
	var componentMetadata []daprcomponents.MetadataItem
	for k, v := range properties {
		value, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("component metadata encoding failed: %s", err)
		}
		componentMetadata = append(componentMetadata, daprcomponents.MetadataItem{
			Name: k,
			Value: daprcomponents.DynamicValue{
				JSON: apiextensionsv1.JSON{Raw: value},
			},
		})
	}

	component := &daprcomponents.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterStream.Name,
			Namespace: clusterStream.Namespace,
		},
		Spec: daprcomponents.ComponentSpec{
			Type:     componentType,
			Version:  "v1",
			Metadata: componentMetadata,
		},
	}

	log.Printf("Created component: %#v", component)

	if err := ctrl.SetControllerReference(clusterStream, component, r.Scheme); err != nil {
		return nil, err
	}
	return component, nil
}

func (r *StreamReconciler) createDaprSubscription(stream *streamingruntime.Stream) (*daprsubscriptions.Subscription, error) {
	// Prepare subscription object for saving into statestore
	route := stream.Spec.Route
	if route == "" {
		route = fmt.Sprintf("/%s", stream.Spec.Topic)
	}

	sub := &daprsubscriptions.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stream.Name,
			Namespace: stream.Namespace,
		},
		Spec: daprsubscriptions.SubscriptionSpec{
			Pubsubname: stream.Spec.ClusterStream,
			Topic:      stream.Spec.Topic,
			Metadata:   nil,
			Routes: daprsubscriptions.Routes{
				Default: route,
			},
		},
		Scopes: stream.Spec.Recipients,
	}

	if err := ctrl.SetControllerReference(stream, sub, r.Scheme); err != nil {
		return nil, err
	}

	return sub, nil
}

func (r *StreamReconciler) updateDaprSubscription(stream *streamingruntime.Stream, sub *daprsubscriptions.Subscription) {
	sub.Spec.Topic = stream.Spec.Topic
	sub.Spec.Pubsubname = stream.Spec.ClusterStream
	sub.Spec.Routes.Default = stream.Spec.Route
	sub.Scopes = stream.Spec.Recipients
}
