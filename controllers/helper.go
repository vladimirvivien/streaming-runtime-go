package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	daprcomponents "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	daprsubscriptions "github.com/dapr/dapr/pkg/apis/subscriptions/v2alpha1"
	streamingruntime "github.com/vladimirvivien/streaming-runtime/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	// establish object ownership
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

	// establish object ownership
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

func (r *ProcessorReconciler) createProcessorDeployment(proc *streamingruntime.Processor) (*appsv1.Deployment, error) {
	replicas := proc.Spec.Replicas
	if replicas == 0 {
		replicas = 1
	}

	// add port info
	proc.Spec.Container.Ports = append(proc.Spec.Container.Ports, corev1.ContainerPort{
		Name:          "app-port",
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

	// establish ownership
	if err := ctrl.SetControllerReference(proc, deployment, r.Scheme); err != nil {
		return nil, err
	}

	return deployment, nil
}

func (r *ProcessorReconciler) updateProcessorDeployment(proc *streamingruntime.Processor, dep *appsv1.Deployment) {
	replicas := proc.Spec.Replicas
	if replicas == 0 {
		replicas = 1
	}
	dep.Spec.Replicas = &replicas
	dep.Spec.Template.Spec.Containers = []corev1.Container{proc.Spec.Container}
}

const defaultJoinerImage = "ghcr.io/vladimirvivien/streaming-runtime/components/joiner:latest"

func (r *JoinerReconciler) createJoinerDeployment(ctx context.Context, joiner *streamingruntime.Joiner) (*appsv1.Deployment, error) {
	var replicas int32 = 1

	// resolve container
	// if not provided, use default image above.
	var container corev1.Container
	if joiner.Spec.Container == nil {
		container = corev1.Container{
			Name:            joiner.Name,
			Image:           defaultJoinerImage,
			ImagePullPolicy: corev1.PullAlways,
		}
	} else {
		container = *joiner.Spec.Container
	}

	// add service port to container
	container.Ports = append(container.Ports, corev1.ContainerPort{
		Name:          "app-port",
		ContainerPort: joiner.Spec.ServicePort,
	})

	// validate and set env data
	if len(joiner.Spec.Streams) < 2 {
		return nil, fmt.Errorf("joiner requires 2 or more streams")
	}

	streamInfo, err := r.collateStreamInfo(ctx, joiner)
	if err != nil {
		return nil, err
	}
	container.Env = []corev1.EnvVar{
		{Name: "JOINER_NAMESPACE", Value: fmt.Sprintf(":%s", joiner.Namespace)},
		{Name: "JOINER_SERVICE_PORT", Value: fmt.Sprintf(":%d", joiner.Spec.ServicePort)},
		{Name: "JOINER_STREAMS_INFO", Value: strings.Join(streamInfo, ";")},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      joiner.Name,
			Namespace: joiner.Namespace,
			Labels:    map[string]string{"app": joiner.Name},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": joiner.Name},
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": joiner.Name},
					Annotations: map[string]string{
						"dapr.io/enabled":  "true",
						"dapr.io/app-id":   joiner.Name,
						"dapr.io/app-port": fmt.Sprintf("%d", joiner.Spec.ServicePort),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}

	// establish ownership
	if err := ctrl.SetControllerReference(joiner, deployment, r.Scheme); err != nil {
		return nil, err
	}

	return deployment, nil
}

func (r *JoinerReconciler) updateJoinerDeployment(joiner *streamingruntime.Joiner, dep *appsv1.Deployment) {
	var container corev1.Container
	if joiner.Spec.Container == nil {
		container = corev1.Container{
			Name:            joiner.Name,
			Image:           defaultJoinerImage,
			ImagePullPolicy: corev1.PullAlways,
		}
	} else {
		container = *joiner.Spec.Container
	}
	dep.Spec.Template.Spec.Containers = []corev1.Container{container}
}

// collateStreamInfo returns Stream info as a []string
// where each element is ClusterStream|Topic|Route
func (r *JoinerReconciler) collateStreamInfo(ctx context.Context, joiner *streamingruntime.Joiner) ([]string, error) {
	var result []string
	for _, streamName := range joiner.Spec.Streams {
		stream := new(streamingruntime.Stream)
		err := r.Get(ctx, types.NamespacedName{Namespace: joiner.Namespace, Name: streamName}, stream)
		if err != nil {
			log.Printf("JoinerReconciler: failed to retrieve Stream: %s", err)
			return nil, err
		}
		route := stream.Spec.Route
		if route == "" {
			route = stream.Spec.Topic
		}
		result = append(result, fmt.Sprintf("%s|%s|%s", stream.Spec.ClusterStream, stream.Spec.Topic, route))
	}
	return result, nil
}
