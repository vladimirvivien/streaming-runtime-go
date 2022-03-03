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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProcessorSpec defines the desired state of Processor
type ProcessorSpec struct {
	Replicas    int32            `json:"replicas"`
	ServicePort int32            `json:"servicePort"`
	Container   corev1.Container `json:"container"`
}

// ProcessorStatus defines the observed state of Processor
type ProcessorStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Processor is the Schema for the processors API
type Processor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProcessorSpec   `json:"spec,omitempty"`
	Status ProcessorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProcessorList contains a list of Processor
type ProcessorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Processor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Processor{}, &ProcessorList{})
}
