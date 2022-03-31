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
	"encoding/json"
	"fmt"
	"strings"

	daprcomponents "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	streamingruntime "github.com/vladimirvivien/streaming-runtime/api/v1alpha1"
)

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

	// Add associated dapr component if not found
	component := new(daprcomponents.Component)
	err = r.Get(ctx, types.NamespacedName{Name: cs.Name, Namespace: cs.Namespace}, component)
	if err != nil && errors.IsNotFound(err) {
		// New component
		componentType := fmt.Sprintf("pubsub.%s", cs.Spec.Protocol)
		component, err = r.createClusterStream(ctx, componentType, cs)
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

		log.Info("Pubsub component created successfully", "Component.Name", component.Name, "Component.Namespace", component.Namespace)
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		log.Error(err, "Failed to get Component", "Component.Name", cs.Name, "Component.Namespace", cs.Namespace)
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
		log.Error(statErr, "Failed to update ClusterStream status")
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

// createClusterStream is a helper method to create v1alpha1 Dapr Component API objects
func (r *ClusterStreamReconciler) createClusterStream(ctx context.Context, componentType string, clusterStream *streamingruntime.ClusterStream) (*daprcomponents.Component, error) {
	log := log.FromContext(ctx)
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

	log.Info("Component created successfully", "Component.Name", component.Name, "Component.Namespace", component.Namespace)

	// establish object ownership
	if err := ctrl.SetControllerReference(clusterStream, component, r.Scheme); err != nil {
		return nil, err
	}
	return component, nil
}
