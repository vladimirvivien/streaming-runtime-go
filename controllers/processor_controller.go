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

// ProcessorReconciler reconciles a Processor object
type ProcessorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=streaming.vivien.io,resources=processors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=processors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=processors/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *ProcessorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	proc := new(streamingruntime.Processor)
	err := r.Get(ctx, req.NamespacedName, proc)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Processor not found, ignoring", "Name", req.Name, "Namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to fetch Processor", "Name", req.Name, "Namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	// is item being deleted?
	if !proc.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil // do nothing, stop reconciliation
	}

	// Retrieve processor service deployment component
	// if not found, create it
	deployment := new(appsv1.Deployment)
	err = r.Get(ctx, types.NamespacedName{Name: proc.Name, Namespace: proc.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating deployment for Processor",
			"Name", proc.Name,
			"Namespace", proc.Namespace,
			"ServicePort", proc.Spec.ServicePort,
		)

		deployment, err = r.createDeployment(proc)
		if err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "Failed to create deployment for Processor",
				"Name", proc.Name,
				"Namespace", proc.Namespace,
				"ServicePort", proc.Spec.ServicePort,
			)
			return ctrl.Result{}, err
		}

		log.Info("Created deployment for Processor successfully",
			"Name", proc.Name,
			"Namespace", proc.Namespace,
			"ServicePort", proc.Spec.ServicePort,
			"Deployment", deployment.Name,
		)
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Processor deployment", "Name", proc.Name, "Namespace", proc.Namespace)
		return ctrl.Result{}, err
	}

	// Update processor
	if err := r.Update(ctx, proc); err != nil {
		log.Error(err, "Failed to update Processor",
			"Namespace", proc.Namespace,
			"Name", proc.Name)
		return ctrl.Result{}, err
	} else {
		// Update deployment
		err = r.Get(ctx, types.NamespacedName{Name: proc.Name, Namespace: proc.Namespace}, deployment)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info("Deployment object not found for processor, ignoring update", "Name", req.Name, "Namespace", req.Namespace)
				return ctrl.Result{}, nil
			}
			log.Error(err, "Failed to get deployment for Processor", "Name", proc.Name, "Namespace", proc.Namespace)
			return ctrl.Result{}, err
		}

		r.updateDeployment(proc, deployment)
		if err := r.Update(ctx, deployment); err != nil {
			log.Error(err, "Failed to update deployment for Processor",
				"Namespace", proc.Namespace,
				"Name", proc.Name,
			)
			return ctrl.Result{}, err
		}
	}

	log.Info("Updated deployment for Processor",
		"Namespace", proc.Namespace,
		"Name", proc.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProcessorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&streamingruntime.Processor{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
