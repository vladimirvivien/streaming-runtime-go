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
	defaultJoinerImage = "ghcr.io/vladimirvivien/streaming-runtime/components/joiner:latest"
)

// JoinerReconciler reconciles a Joiner object
type JoinerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=streaming.vivien.io,resources=joiners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=joiners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=joiners/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=streams,verbs=get;list;watch;create;update;patch;delete

func (r *JoinerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Attempt to find existing object
	joiner := new(streamingruntime.Joiner)
	err := r.Get(ctx, req.NamespacedName, joiner)
	// if object not found, then create it
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Joiner not found, ignoring", "Name", req.Name, "Namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to fetch Joiner", "Name", req.Name, "Namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	// Is object being deleted?
	if !joiner.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil // do nothing, stop reconciliation
	}

	// Retrieve joiner deployment component
	// if not found, create it
	deployment := new(appsv1.Deployment)
	err = r.Get(ctx, types.NamespacedName{Name: joiner.Name, Namespace: joiner.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating deployment for Joiner",
			"Name", joiner.Name,
			"Namespace", joiner.Namespace,
		)

		deployment, err = r.createJoinerDeployment(ctx, joiner)
		if err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "Failed to create deployment for Joiner",
				"Name", joiner.Name,
				"Namespace", joiner.Namespace,
			)
			return ctrl.Result{}, err
		}

		log.Info("Created deployment for Joiner successfully",
			"Name", joiner.Name,
			"Namespace", joiner.Namespace,
			"Deployment", deployment.Name,
		)
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Joiner deployment", "Name", joiner.Name, "Namespace", joiner.Namespace)
		return ctrl.Result{}, err
	}

	log.Info("Updated deployment for Joiner",
		"Namespace", joiner.Namespace,
		"Name", joiner.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JoinerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&streamingruntime.Joiner{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

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
	if len(joiner.Spec.Streams) != 2 {
		return nil, fmt.Errorf("joiner must have 2 input streams")
	}

	if joiner.Spec.Target == "" {
		return nil, fmt.Errorf("joiner missing valid target")
	}

	streamInfo, err := r.collateStreamInfo(ctx, joiner)
	if err != nil {
		return nil, err
	}

	container.Env = []corev1.EnvVar{
		{Name: "JOINER_SERVICE_PORT", Value: fmt.Sprintf(":%d", joiner.Spec.ServicePort)},
		{Name: "JOINER_STREAM0_INFO", Value: streamInfo[0]},
		{Name: "JOINER_STREAM1_INFO", Value: streamInfo[1]},
		{Name: "JOINER_TARGET", Value: validateTarget(joiner.Spec.Target)},
		{Name: "JOINER_WINDOW_SIZE", Value: joiner.Spec.Window},
	}

	// Setup data selection
	if joiner.Spec.Select != nil {
		container.Env = append(
			container.Env,
			corev1.EnvVar{Name: "JOINER_SELECT_FILTER_EXPRESSION", Value: joiner.Spec.Select.Where},
			corev1.EnvVar{Name: "JOINER_SELECT_DATA_EXPRESSION", Value: joiner.Spec.Select.Data},
		)
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
