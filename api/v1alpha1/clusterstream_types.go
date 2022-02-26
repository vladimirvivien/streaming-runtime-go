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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DaprComponentType string

var (
	KafkaProtocolName = "kafka"
)

// ClusterStreamSpec defines the desired state of ClusterStream
type ClusterStreamSpec struct {
	Protocol   string            `json:"protocol"`
	Properties map[string]string `json:"properties"`
	AuthType   string            `json:"authType,omitempty"`
}

// ClusterStreamStatus defines the observed state of ClusterStream
type ClusterStreamStatus struct {
	Provider string `json:"provider"`
	Servers  string `json:"servers"`
	Status   string `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterStream is the Schema for the clusterstreams API
type ClusterStream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterStreamSpec   `json:"spec,omitempty"`
	Status ClusterStreamStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterStreamList contains a list of ClusterStream
type ClusterStreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterStream `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterStream{}, &ClusterStreamList{})
}
