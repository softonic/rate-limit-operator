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

package istio_v1alpha3

import (
	//"github.com/gogo/protobuf/types"
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NOTE: This type is used only to have a VirtualService runtime.Object (Istio's is not). we are only interested in
// Hosts property. This is not going to be used as CR, could be moved somewherelese

type WorkloadSelector struct {
	// One or more labels that indicate a specific set of pods/VMs
	// on which the configuration should be applied. The scope of
	// label search is restricted to the configuration namespace in which the
	// the resource is present.
	Labels map[string]string `json:"labels,omitempty"`
}

type ApplyTo string

type PatchContext string

type ProxyMatch struct {
	ProxyVersion string            `json:"proxyVersion,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type ListenerMatch_SubFilterMatch struct {
	// The filter name to match on.
	Name string `json:"name,omitempty"`
}

type ListenerMatch_FilterMatch struct {
	// The filter name to match on.
	// For standard Envoy filters, canonical filter names should be used.
	// Refer to https://www.envoyproxy.io/docs/envoy/latest/version_history/v1.14.0#deprecated for canonical names.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// The next level filter within this filter to match
	// upon. Typically used for HTTP Connection Manager filters and
	// Thrift filters.
	SubFilter ListenerMatch_SubFilterMatch `json:"sub_filter,omitempty"`
}

type ListenerMatch_FilterChainMatch struct {
	// The name assigned to the filter chain.
	Name string `json:"name,omitempty"`
	// The SNI value used by a filter chain's match condition.  This
	// condition will evaluate to false if the filter chain has no
	// sni match.
	Sni string `json:"sni,omitempty"`
	// Applies only to SIDECAR_INBOUND context. If non-empty, a
	// transport protocol to consider when determining a filter
	// chain match.  This value will be compared against the
	// transport protocol of a new connection, when it's detected by
	// the tls_inspector listener filter.
	//
	// Accepted values include:
	//
	// * `raw_buffer` - default, used when no transport protocol is detected.
	// * `tls` - set when TLS protocol is detected by the TLS inspector.
	TransportProtocol string `json:"transport_protocol,omitempty"`
	// Applies only to sidecars. If non-empty, a comma separated set
	// of application protocols to consider when determining a
	// filter chain match.  This value will be compared against the
	// application protocols of a new connection, when it's detected
	// by one of the listener filters such as the http_inspector.
	//
	// Accepted values include: h2,http/1.1,http/1.0
	ApplicationProtocols string `json:"application_protocols,omitempty"`
	// The name of a specific filter to apply the patch to. Set this
	// to envoy.filters.network.http_connection_manager to add a filter or apply a
	// patch to the HTTP connection manager.
	Filter ListenerMatch_FilterMatch `json:"filter,omitempty"`
}

type ListenerMatch struct {
	// The service port/gateway port to which traffic is being
	// sent/received. If not specified, matches all listeners. Even though
	// inbound listeners are generated for the instance/pod ports, only
	// service ports should be used to match listeners.
	PortNumber uint32 `json:"port_number,omitempty"`
	// Instead of using specific port numbers, a set of ports matching
	// a given service's port name can be selected. Matching is case
	// insensitive.
	// Not implemented.
	// $hide_from_docs
	PortName string `json:"port_name,omitempty"`
	// Match a specific filter chain in a listener. If specified, the
	// patch will be applied to the filter chain (and a specific
	// filter if specified) and not to other filter chains in the
	// listener.
	FilterChain ListenerMatch_FilterChainMatch `json:"filter_chain,omitempty"`
	// Match a specific listener by its name. The listeners generated
	// by Pilot are typically named as IP:Port.
	Name string `json:"name,omitempty"`
}

type EnvoyConfigObjectMatch_Listener struct {
	Listener ListenerMatch `json:"listener,omitempty"`
}

type ClusterMatch struct {
	// The service port for which this cluster was generated.  If
	// omitted, applies to clusters for any port.
	PortNumber uint32 `json:"port_number,omitempty"`
	// The fully qualified service name for this cluster. If omitted,
	// applies to clusters for any service. For services defined
	// through service entries, the service name is same as the hosts
	// defined in the service entry.
	Service string `json:"service,omitempty"`
	// The subset associated with the service. If omitted, applies to
	// clusters for any subset of a service.
	Subset string `json:"subset,omitempty"`
	// The exact name of the cluster to match. To match a specific
	// cluster by name, such as the internally generated "Passthrough"
	// cluster, leave all fields in clusterMatch empty, except the
	// name.
	Name string `json:"name,omitempty"`
}

type RouteConfigurationMatch_RouteMatch_Action string

type RouteConfigurationMatch_RouteMatch struct {
	// The Route objects generated by default are named as
	// "default".  Route objects generated using a virtual service
	// will carry the name used in the virtual service's HTTP
	// routes.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Match a route with specific action type.
	Action RouteConfigurationMatch_RouteMatch_Action `json:"action,omitempty"`
}

type RouteConfigurationMatch_VirtualHostMatch struct {
	// The VirtualHosts objects generated by Istio are named as
	// host:port, where the host typically corresponds to the
	// VirtualService's host field or the hostname of a service in the
	// registry.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Match a specific route within the virtual host.
	Route RouteConfigurationMatch_RouteMatch `json:"route,omitempty"`
}

type RouteConfigurationMatch struct {
	// The service port number or gateway server port number for which
	// this route configuration was generated. If omitted, applies to
	// route configurations for all ports.
	PortNumber uint32 `json:"port_number,omitempty"`
	// Applicable only for GATEWAY context. The gateway server port
	// name for which this route configuration was generated.
	PortName string `json:"port_name,omitempty"`
	// The Istio gateway config's namespace/name for which this route
	// configuration was generated. Applies only if the context is
	// GATEWAY. Should be in the namespace/name format. Use this field
	// in conjunction with the portNumber and portName to accurately
	// select the Envoy route configuration for a specific HTTPS
	// server within a gateway config object.
	Gateway string `json:"gateway,omitempty"`
	// Match a specific virtual host in a route configuration and
	// apply the patch to the virtual host.
	Vhost RouteConfigurationMatch_VirtualHostMatch `protobuf:"bytes,4,opt,name=vhost,proto3" json:"vhost,omitempty"`
	// Route configuration name to match on. Can be used to match a
	// specific route configuration by name, such as the internally
	// generated "http_proxy" route configuration for all sidecars.
	Name string `protobuf:"bytes,5,opt,name=name,proto3" json:"name,omitempty"`
}

type RouteConfiguration struct {
	RouteConfiguration RouteConfigurationMatch `protobuf:"bytes,4,opt,name=route_configuration,json=routeConfiguration,proto3,oneof"`
}

type EnvoyConfigObjectMatch struct {
	Context string     `json:"context,omitempty"`
	Proxy   ProxyMatch `json:"proxy,omitempty"`

	// Types that are valid to be assigned to ObjectTypes:
	//	*EnvoyFilter_EnvoyConfigObjectMatch_Listener
	//	*EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration
	//	*EnvoyConfigObjectMatch_Cluster
	//ObjectTypes          EnvoyConfigObjectMatch_Listener `protobuf_oneof:"object_types"`
	//ObjectTypes          isEnvoyFilter_EnvoyConfigObjectMatch_ObjectTypes `protobuf_oneof:"object_types"`
	Listener           *ListenerMatch           `json:"listener,omitempty"`
	Cluster            *ClusterMatch            `json:"cluster,omitempty"`
	RouteConfiguration *RouteConfigurationMatch `json:"routeConfiguration,omitempty"`
}

type Patch_Operation string

type Patch_FilterClass int32

//type Values json.RawMessage

type Patch struct {
	Operation   string            `json:"operation,omitempty"`
	Value       json.RawMessage   `json:"value,omitempty"`
	FilterClass Patch_FilterClass `json:"filterClass,omitempty"`
}

type EnvoyConfigObjectPatch struct {
	ApplyTo string                 `json:"applyTo,omitempty"`
	Match   EnvoyConfigObjectMatch `json:"match,omitempty"`
	Patch   Patch                  `json:"patch,omitempty"`
}

// EnvoyFilterSpec defines the desired state of EnvoyFilter
type EnvoyFilterSpec struct {
	WorkloadSelector WorkloadSelector         `json:"workloadSelector,omitempty"`
	ConfigPatches    []EnvoyConfigObjectPatch `json:"configPatches,omitempty"`
}

// EnvoyFilterStatus defines the observed state of EnvoyFilter
type EnvoyFilterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// EnvoyFilter is the Schema for the EnvoyFilters API
type EnvoyFilter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvoyFilterSpec   `json:"spec,omitempty"`
	Status EnvoyFilterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvoyFilterList contains a list of EnvoyFilter
type EnvoyFilterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvoyFilter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvoyFilter{}, &EnvoyFilterList{})
}
