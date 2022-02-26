package controllers

import (
	"encoding/json"
	"fmt"
	"log"

	daprcomponents "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	streamingruntime "github.com/vmware-tanzu/streaming-runtime/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getMetadataStringValues(keyName string, properties map[string]interface{}) (result []string) {
	for key, value := range properties {
		if key == keyName {
			data, ok := value.(string)
			if !ok {
				result = append(result, "<unknown>")
				continue
			}
			result = append(result, data)
		}
	}
	return
}

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
			Metadata: componentMetadata,
		},
	}

	log.Printf("Created component: %#v", component)

	if err := ctrl.SetControllerReference(clusterStream, component, r.Scheme); err != nil {
		return nil, err
	}
	return component, nil
}
