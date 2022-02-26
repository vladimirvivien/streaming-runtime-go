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
	"strings"

	daprcomponents "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	streamingruntime "github.com/vladimirvivien/streaming-runtime/api/v1alpha1"
)

//var (
//	finalizer = "clusterstream.streaming.vivien.io/finalizer"
//)

// ClusterStreamReconciler reconciles a ClusterStream object
type ClusterStreamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=streaming.vivien.io,resources=clusterstreams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=clusterstreams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=streaming.vivien.io,resources=clusterstreams/finalizers,verbs=update
//+kubebuilder:rbac:groups=dapr.io,resources=components,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterStreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx) // get logger from context

	cs := new(streamingruntime.ClusterStream)
	err := r.Get(ctx, req.NamespacedName, cs)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			log.Info("ClusterStream not found, ignoring", "Name", req.Name, "Namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to fetch ClusterStream", "Name", req.Name, "Namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	// Object deletion
	// Examine DeletionTimestamp to determine if object is under deletion
	//if cs.ObjectMeta.DeletionTimestamp.IsZero() {
	//	// The object is not being deleted, register finalizer for this reconciler.
	//	if !controllerutil.ContainsFinalizer(&cs, finalizer) {
	//		controllerutil.AddFinalizer(&cs, finalizer)
	//		if err := r.Update(ctx, &cs); err != nil {
	//			return ctrl.Result{}, err
	//		}
	//	}
	//} else {
	//	// The object is being deleted
	//	if controllerutil.ContainsFinalizer(&cs, finalizer) {
	//		if err := r.Delete(ctx, &cs); err != nil {
	//			return ctrl.Result{}, err
	//		}
	//
	//		// remove our finalizer, update object for eventual garbage collection
	//		controllerutil.RemoveFinalizer(&cs, finalizer)
	//		if err := r.Update(ctx, &cs); err != nil {
	//			return ctrl.Result{}, err
	//		}
	//	}
	//
	//	// Stop reconciliation as the item is being deleted
	//	return ctrl.Result{}, nil
	//}

	// Add associated dapr component if not found
	component := new(daprcomponents.Component)
	err = r.Get(ctx, types.NamespacedName{Name: cs.Name, Namespace: cs.Namespace}, component)
	if err != nil && errors.IsNotFound(err) {
		// New component
		componentType := fmt.Sprintf("%s.pubsub", cs.Spec.Protocol)
		component, err = r.createDaprComponent(componentType, cs)
		if err != nil {
			return ctrl.Result{}, err
		}

		log.V(1).Info("Creating pub/sub dapr component for ClusterStream",
			"Namespace", component.Namespace,
			"Name", component.Name,
			"Type", componentType)

		if err := r.Create(ctx, component); err != nil {
			log.Error(err, "Failed to create pub/sub dapr component for ClusterStream",
				"Component.Namespace", component.Namespace,
				"Component.Name", component.Name,
				"Component.Type", componentType)
			return ctrl.Result{}, err
		}

		log.Info("Component created successfully", "Component.Name", component.Name, "Component.Namespace", component.Namespace)
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Component", "Component.Name", cs.Name, "Component.Namespace", cs.Namespace)
		return ctrl.Result{}, err
	}

	// if associated dapr component is found, update it
	if err := r.Update(ctx, component); err != nil {
		log.Error(err, "Failed to create pub/sub dapr component",
			"Namespace", component.Namespace,
			"Name", component.Name,
			"Type", component.Spec.Type)
		return ctrl.Result{}, err
	}

	// Update status for the
	cs.Status.Provider = streamingruntime.KafkaProtocolName
	brokers := getMetadataStringValues("brokers", cs.Spec.Properties)

	if len(brokers) == 0 {
		cs.Status.Status = "Error"
		err = fmt.Errorf("kafka: missing bootstrap servers")
	} else {
		cs.Status.Status = "Ready"
		cs.Status.Servers = strings.Join(brokers, ",")
	}

	if statErr := r.Status().Update(ctx, cs); statErr != nil {
		log.Error(statErr, "Failed to update status")
		return ctrl.Result{}, statErr
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterStreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&streamingruntime.ClusterStream{}).
		Owns(&daprcomponents.Component{}).
		Complete(r)
}
