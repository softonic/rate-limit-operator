/*


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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RateLimitDimension map[string]string

type Dimensions map[string]RateLimitDimension

type DimensionsList []Dimensions

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RateLimitSpec defines the desired state of RateLimit
type RateLimitSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	TargetRef          v1.ObjectReference `json:"targetRef"`
	DestinationCluster string             `json:"destinationCluster"`
	Unit               string             `json:"unit"`
	RequestsPerUnit    uint32             `json:"requestPerUnit"`
	Dimensions         DimensionsList     `json:"dimensions"`
	WorkloadSelector   map[string]string  `json:"workloadselector"`
}

// RateLimitStatus defines the observed state of RateLimit
type RateLimitStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// RateLimit is the Schema for the ratelimits API
type RateLimit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RateLimitSpec   `json:"spec,omitempty"`
	Status RateLimitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RateLimitList contains a list of RateLimit
type RateLimitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RateLimit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RateLimit{}, &RateLimitList{})
}
