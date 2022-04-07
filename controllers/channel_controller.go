/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	streamingruntime "github.com/vladimirvivien/streaming-runtime/api/v1alpha1"
)

const (
	defaultChannelImage = "ghcr.io/vladimirvivien/streaming-runtime/components/channel:latest"
)

// ChannelReconciler reconciles a Channel object
type ChannelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=streaming.vivien.io,resources=channels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=channels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=channels/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *ChannelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Attempt to find existing object
	channel := new(streamingruntime.Channel)
	err := r.Get(ctx, req.NamespacedName, channel)
	// if object not found, then create it
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Channel not found, ignoring", "Name", req.Name, "Namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to fetch Channel", "Name", req.Name, "Namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	// Is object being deleted?
	if !channel.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil // do nothing, stop reconciliation
	}

	// Retrieve joiner deployment component
	// if not found, create it
	deployment := new(appsv1.Deployment)
	err = r.Get(ctx, types.NamespacedName{Name: channel.Name, Namespace: channel.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating deployment for Joiner",
			"Name", channel.Name,
			"Namespace", channel.Namespace,
		)

		deployment, err = r.createChanDeployment(ctx, channel)
		if err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "Failed to create deployment for Channel",
				"Name", channel.Name,
				"Namespace", channel.Namespace,
			)
			return ctrl.Result{}, err
		}

		log.Info("Created deployment for Channel successfully",
			"Name", channel.Name,
			"Namespace", channel.Namespace,
			"Deployment", deployment.Name,
		)
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Joiner deployment", "Name", channel.Name, "Namespace", channel.Namespace)
		return ctrl.Result{}, err
	}

	log.Info("Updated deployment for Channel",
		"Namespace", channel.Namespace,
		"Name", channel.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChannelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&streamingruntime.Channel{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *ChannelReconciler) createChanDeployment(ctx context.Context, channel *streamingruntime.Channel) (*appsv1.Deployment, error) {
	var replicas int32 = 1

	// resolve container
	// if not provided, use default image above.
	var container corev1.Container
	if channel.Spec.Container == nil {
		container = corev1.Container{
			Name:            channel.Name,
			Image:           defaultChannelImage,
			ImagePullPolicy: corev1.PullAlways,
		}
	} else {
		container = *channel.Spec.Container
	}

	// add service port to container
	container.Ports = append(container.Ports, corev1.ContainerPort{
		Name:          "app-port",
		ContainerPort: channel.Spec.ServicePort,
	})

	if channel.Spec.Stream.To[0].Stream == "" && channel.Spec.Stream.To[0].Component == "" {
		return nil, fmt.Errorf("channel stream.To must have a stream or a component specified")
	}

	mode := channel.Spec.Mode
	if mode == "" || (mode != "stream" && mode != "aggregate") {
		mode = "stream"
	}
	if mode == "aggregate" && channel.Spec.Trigger == "" {
		return nil, fmt.Errorf("channel missing aggregate trigger expression")
	}

	container.Env = []corev1.EnvVar{
		{Name: "CHANNEL_SERVICE_PORT", Value: fmt.Sprintf(":%d", channel.Spec.ServicePort)},
		{Name: "CHANNEL_MODE", Value: mode},
		{Name: "CHANNEL_AGGREGATE_TRIGGER", Value: channel.Spec.Trigger},
		{Name: "CHANNEL_STREAM_FROM", Value: channel.Spec.Stream.From[0]},
		{Name: "CHANNEL_STREAM_TO_STREAM", Value: channel.Spec.Stream.To[0].Stream},
		{Name: "CHANNEL_STREAM_TO_COMPONENT", Value: channel.Spec.Stream.To[0].Component},
		{Name: "CHANNEL_STREAM_WHERE", Value: channel.Spec.Stream.Where},
		{Name: "CHANNEL_STREAM_SELECT", Value: channel.Spec.Stream.Select},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      channel.Name,
			Namespace: channel.Namespace,
			Labels:    map[string]string{"app": channel.Name},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": channel.Name},
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": channel.Name},
					Annotations: map[string]string{
						"dapr.io/enabled":  "true",
						"dapr.io/app-id":   channel.Name,
						"dapr.io/app-port": fmt.Sprintf("%d", channel.Spec.ServicePort),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}

	// establish ownership
	if err := ctrl.SetControllerReference(channel, deployment, r.Scheme); err != nil {
		return nil, err
	}

	return deployment, nil
}
