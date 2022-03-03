package controllers

import (
	"encoding/json"
	"fmt"
	"log"

	daprcomponents "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	daprsubscriptions "github.com/dapr/dapr/pkg/apis/subscriptions/v2alpha1"
	streamingruntime "github.com/vladimirvivien/streaming-runtime/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

func (r *ProcessorReconciler) createDeployment(proc *streamingruntime.Processor) (*appsv1.Deployment, error) {
	replicas := proc.Spec.Replicas
	if replicas == 0 {
		replicas = 1
	}

	// add port info
	proc.Spec.Container.Ports = append(proc.Spec.Container.Ports, corev1.ContainerPort{
		Name:          "streamrt-app-port",
		HostPort:      0,
		ContainerPort: proc.Spec.ServicePort,
	})

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      proc.Name,
			Namespace: proc.Namespace,
			Labels:    map[string]string{"app": proc.Name},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": proc.Name},
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": proc.Name},
					Annotations: map[string]string{
						"dapr.io/enabled":  "true",
						"dapr.io/app-id":   proc.Name,
						"dapr.io/app-port": fmt.Sprintf("%d", proc.Spec.ServicePort),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{proc.Spec.Container},
				},
			},
		},
	}

	return deployment, nil
}

func (r *ProcessorReconciler) updateDeployment(proc *streamingruntime.Processor, dep *appsv1.Deployment) {
	dep.Spec.Template.Spec.Containers = []corev1.Container{proc.Spec.Container}
}
