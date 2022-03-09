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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	streamingruntime "github.com/vladimirvivien/streaming-runtime/api/v1alpha1"
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

		deployment, err = r.createJoinerDeployment(joiner)
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
