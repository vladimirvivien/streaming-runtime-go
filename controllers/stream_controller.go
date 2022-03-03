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

	daprsubscriptions "github.com/dapr/dapr/pkg/apis/subscriptions/v2alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	// Look for dapr subscription object
	// if not found, create new one
	sub := new(daprsubscriptions.Subscription)
	err = r.Get(ctx, types.NamespacedName{Name: stream.Name, Namespace: stream.Namespace}, sub)
	if err != nil && errors.IsNotFound(err) {
		// New Subscription component
		sub, err = r.createDaprSubscription(stream)
		if err != nil {
			return ctrl.Result{}, err
		}

		log.V(1).Info("Creating subscription for Stream",
			"ClusterStream", stream.Spec.ClusterStream,
			"Namespace", stream.Namespace,
			"Name", stream.Name,
			"Topic", stream.Spec.Topic,
		)

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

	// Update subscription if found
	if err := r.Update(ctx, stream); err != nil {
		log.Error(err, "Failed to update Stream object",
			"Namespace", stream.Namespace,
			"Name", stream.Name)
		return ctrl.Result{}, err
	} else {
		// update the dapr.Subscription object as well
		// Get latest dapr.Subscription
		err = r.Get(ctx, types.NamespacedName{Name: stream.Name, Namespace: stream.Namespace}, sub)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info("Subscription object not found for stream, ignoring update", "Name", req.Name, "Namespace", req.Namespace)
				return ctrl.Result{}, nil
			}
			log.Error(err, "Failed to get subscription", "Name", stream.Name, "Namespace", stream.Namespace, "Topic", stream.Spec.Topic)
			return ctrl.Result{}, err
		}

		r.updateDaprSubscription(stream, sub)
		if err := r.Update(ctx, sub); err != nil {
			log.Error(err, "Failed to update subscription for Stream",
				"ClusterStream", stream.Spec.ClusterStream,
				"Namespace", stream.Namespace,
				"Name", stream.Name,
				"Topic", stream.Spec.Topic)
			return ctrl.Result{}, err
		}
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
