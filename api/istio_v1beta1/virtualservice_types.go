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

package istio_v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NOTE: This type is used only to have a VirtualService runtime.Object (Istio's is not). we are only interested in
// Hosts property. This is not going to be used as CR, could be moved somewherelese

type Destination struct {
	Host   string `json:"host,omitempty"`
	Subset string `json:"subset,omitempty"`
}

type HTTPRouteDestination struct {
	Destination *Destination `json:"destination,omitempty"`
}

type HTTPRoute struct {
	Route []*HTTPRouteDestination `json:"route,omitempty"`
}

// VirtualServiceSpec defines the desired state of VirtualService
type VirtualServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Gateways []string     `json:"gateways,omitempty"`
	Hosts    []string     `json:"hosts,omitempty"`
	Http     []*HTTPRoute `json:"http,omitempty"`
}

// VirtualServiceStatus defines the observed state of VirtualService
type VirtualServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// VirtualService is the Schema for the virtualservices API
type VirtualService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualServiceSpec   `json:"spec,omitempty"`
	Status VirtualServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualServiceList contains a list of VirtualService
type VirtualServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualService{}, &VirtualServiceList{})
}
