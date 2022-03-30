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

// JoinerSpec defines the desired state of Joiner
type JoinerSpec struct {
	ServicePort int32    `json:"servicePort"`
	Streams     []string `json:"streams"`
	Window      string   `json:"window"`
	Target      string   `json:"target"`
	// +optional
	FilterExpression string `json:"filterExpression"`
	// +optional
	DataExpression string `json:"dataExpression"`
	// +optional
	Select *DataSelection `json:"select,omitempty"`
	// +optional
	Container *corev1.Container `json:"container"`
}

// DataSelection defines data selection expressions to select and filter out streamed events
type DataSelection struct {
	// +optional
	Data string `json:"data"`
	// +optional
	Where string `json:"where"`
}

// JoinerStatus defines the observed state of Joiner
type JoinerStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Joiner is the Schema for the joiners API
type Joiner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JoinerSpec   `json:"spec,omitempty"`
	Status JoinerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// JoinerList contains a list of Joiner
type JoinerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Joiner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Joiner{}, &JoinerList{})
}
