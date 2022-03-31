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

	daprsubscriptions "github.com/dapr/dapr/pkg/apis/subscriptions/v2alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	streamingruntime "github.com/vladimirvivien/streaming-runtime/api/v1alpha1"
)

// StreamReconciler reconciles a Stream object
type StreamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=streaming.vivien.io,resources=streams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=streams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=streams/finalizers,verbs=update
//+kubebuilder:rbac:groups=dapr.io,resources=subscriptions,verbs=get;list;watch;create;update;patch;delete

func (r *StreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	stream := new(streamingruntime.Stream)
	err := r.Get(ctx, req.NamespacedName, stream)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			log.Info("Stream not found, ignoring", "Name", req.Name, "Namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to fetch Stream", "Name", req.Name, "Namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	// is item being deleted?
	if !stream.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil // do nothing, stop reconciliation
	}

	// Look for dapr subscription object
	// if not found, create new one
	sub := new(daprsubscriptions.Subscription)
	err = r.Get(ctx, types.NamespacedName{Name: stream.Name, Namespace: stream.Namespace}, sub)
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating subscription for Stream",
			"ClusterStream", stream.Spec.ClusterStream,
			"Namespace", stream.Namespace,
			"Name", stream.Name,
			"Topic", stream.Spec.Topic,
		)

		// New Subscription component
		sub, err = r.createDaprSubscription(ctx, stream)
		if err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, sub); err != nil {
			log.Error(err, "Failed to create subscription for Stream",
				"ClusterStream", stream.Spec.ClusterStream,
				"Namespace", stream.Namespace,
				"Name", stream.Name,
				"Topic", stream.Spec.Topic)
			return ctrl.Result{}, err
		}

		log.Info("Created dapr Subscription successfully",
			"Stream", stream.Name,
			"Pubsub", sub.Spec.Pubsubname,
			"Name", sub.Name,
			"Namespace", sub.Namespace,
			"Topic", sub.Spec.Topic)
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get subscription", "Name", stream.Name, "Namespace", stream.Namespace, "Topic", stream.Spec.Topic)
		return ctrl.Result{}, err
	}

	log.Info("Updated subscription object",
		"Namespace", sub.Name,
		"Name", sub.Name,
		"Topic", sub.Spec.Topic)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&streamingruntime.Stream{}).
		Owns(&daprsubscriptions.Subscription{}).
		Complete(r)
}

func (r *StreamReconciler) createDaprSubscription(_ context.Context, stream *streamingruntime.Stream) (*daprsubscriptions.Subscription, error) {
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
