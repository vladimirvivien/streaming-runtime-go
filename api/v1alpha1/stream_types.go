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

// StreamSpec defines the desired state of Stream
type StreamSpec struct {
	ClusterStream string `json:"clusterStream"`
	Topic         string `json:"topic"`

	// +optional
	Recipients []string `json:"recipients"`
	// +optional
	Route string `json:"route,omitempty"`
	// +optional
	Properties map[string]string `json:"properties,omitempty"`
}

// StreamStatus defines the observed state of Stream
type StreamStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Stream is the Schema for the streams API
type Stream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StreamSpec   `json:"spec,omitempty"`
	Status StreamStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StreamList contains a list of Stream
type StreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stream `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stream{}, &StreamList{})
}
