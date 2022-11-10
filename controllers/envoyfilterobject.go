package controllers

import (
	"encoding/json"
	"os"

	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	//yaml "gopkg.in/yaml.v1"
	"gopkg.in/yaml.v2"
	"k8s.io/klog"

	ratelimitTypes "github.com/softonic/rate-limit-operator/pkg/ratelimit/types"
	networking "istio.io/api/networking/v1alpha3"
	clientIstio "istio.io/client-go/pkg/apis/networking/v1alpha3"

	"strings"
)

type EnvoyFilterObject struct {
	ApplyTo   networking.EnvoyFilter_ApplyTo
	Operation networking.EnvoyFilter_Patch_Operation
	RawConfig string
	//RawConfig             json.RawMessage
	TypeConfigObjectMatch string
	ClusterEndpoint       string
	Context               string
	Labels                map[string]string
	NameVhost             string
	Routes                []string
}

func (r *RateLimitReconciler) prepareUpdateEnvoyFilterObjects(rateLimitInstance networkingv1alpha1.RateLimit, baseName string, controllerNamespace string) error {

	istioNamespace := os.Getenv("ISTIO_NAMESPACE")

	jsonActions := retrieveJsonActions(rateLimitInstance, baseName)

	namespace := rateLimitInstance.Spec.TargetRef.Namespace
	nameVirtualService := rateLimitInstance.Spec.TargetRef.Name

	virtualService, err := r.getVirtualService(namespace, nameVirtualService)
	if err != nil {
		klog.Infof("Virtualservice does not exists. Error: %v", err)
	}

	gatewayIngress := strings.Split(virtualService.Spec.Gateways[0], "/")

	namespaceVirtualService := gatewayIngress[0]
	nameGatewayVirtualService := gatewayIngress[1]

	Gateway, err := r.getGateway(namespaceVirtualService, nameGatewayVirtualService)
	if err != nil {
		klog.Infof("Gateway does not exists")
	}

	gatewaySelector := Gateway.Spec.Selector

	firstElementHosts := strings.Join(virtualService.Spec.Hosts, "")

	nameVhost := firstElementHosts + ":80"

	address := os.Getenv("ADDRESS_RATELIMIT_ENDPOINT")

	fqdn := address + "." + controllerNamespace + ".svc.cluster.local"

	nameCluster := "rate_limit_service_" + baseName

	value, err := createClusterPatchValue(fqdn, nameCluster)
	if err != nil {
		return err
	}

	//payload := []byte(fmt.Sprintf(`{"connect_timeout":"1.25s","load_assignment":{"cluster_name":"%s","endpoints":[{"lb_endpoints":[{"endpoint":{"address":{"socket_address":{"address":"%s","port_value":8081}}}}]}]},"http2_protocol_options":{},"lb_policy":"ROUND_ROBIN","name":"%s","type":"STRICT_DNS"}`, fqdn, fqdn, nameCluster))

	//rawConfigCluster := json.RawMessage(payload)

	//rawConfigCluster := string(payload)

	labels := gatewaySelector

	envoyFilterObjectCluster := EnvoyFilterObject{
		Operation:             networking.EnvoyFilter_Patch_ADD,
		ApplyTo:               networking.EnvoyFilter_CLUSTER,
		RawConfig:             value,
		TypeConfigObjectMatch: "Cluster",
		ClusterEndpoint:       fqdn,
		Labels:                labels,
	}

	envoyFilterClusterDesired := envoyFilterObjectCluster.composeEnvoyFilter(baseName+"-cluster", istioNamespace)

	envoyFilterCluster := &clientIstio.EnvoyFilter{}

	err = r.applyEnvoyFilter(envoyFilterClusterDesired, envoyFilterCluster, baseName+"-cluster", istioNamespace)
	if err != nil {
		klog.Infof("Cannot apply EF")
		return err
	}

	domain := baseName

	value, err = createHttpFilterPatchValue(domain, nameCluster)
	if err != nil {
		return err
	}

	//payload = []byte(fmt.Sprintf(`{"typed_config":{"@type":"type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit","domain":"%s","rate_limit_service":{"transport_api_version": "V3","grpc_service":{"envoy_grpc":{"cluster_name":"%s"},"timeout":"1.25s"}}},"name":"envoy.filters.http.ratelimit"}`, domain, nameCluster))

	//rawConfigHTTPFilter := string(payload)

	envoyFilterObjectListener := EnvoyFilterObject{
		Operation:             networking.EnvoyFilter_Patch_INSERT_BEFORE,
		ApplyTo:               networking.EnvoyFilter_HTTP_FILTER,
		RawConfig:             value,
		TypeConfigObjectMatch: "Listener",
		Context:               "GATEWAY",
		Labels:                labels,
	}

	envoyFilterHTTPFilterDesired := envoyFilterObjectListener.composeEnvoyFilter(baseName+"-envoy-filter", istioNamespace)

	envoyFilterHTTPFilter := &clientIstio.EnvoyFilter{}

	err = r.applyEnvoyFilter(envoyFilterHTTPFilterDesired, envoyFilterHTTPFilter, baseName+"-envoy-filter", istioNamespace)
	if err != nil {
		klog.Infof("Cannot apply EF")
		return err
	}

	initJson := []byte(`{"route":`)
	finalJson := []byte(`}`)

	intermediateJson := append(initJson, jsonActions...)

	rawConfigHTTPRoute := string(append(intermediateJson, finalJson...))
	//rawConfigHTTPRoute := json.RawMessage(append(intermediateJson, finalJson...))

	//rawConfigHTTPRoute := json.RawMessage(`{"route":{"rate_limits":[{"actions":[{"request_headers":{"descriptor_key":"remote_address","header_name":"x-custom-user-ip"}},{"destination_cluster":{}}]}]}}`)

	var routesToApply []string

	if len(rateLimitInstance.Spec.ApplyToRoutes) > 0 {
		routesToApply = rateLimitInstance.Spec.ApplyToRoutes
	}

	envoyFilterObjectRouteConfiguration := EnvoyFilterObject{
		Operation:             networking.EnvoyFilter_Patch_MERGE,
		ApplyTo:               networking.EnvoyFilter_HTTP_ROUTE,
		RawConfig:             rawConfigHTTPRoute,
		TypeConfigObjectMatch: "RouteConfiguration",
		Context:               "GATEWAY",
		Labels:                labels,
		NameVhost:             nameVhost,
		Routes:                routesToApply,
	}

	envoyFilterHTTPRouteDesired := envoyFilterObjectRouteConfiguration.composeEnvoyFilter(baseName+"-route", istioNamespace)

	envoyFilterHTTPRoute := &clientIstio.EnvoyFilter{}

	err = r.applyEnvoyFilter(envoyFilterHTTPRouteDesired, envoyFilterHTTPRoute, baseName+"-route", istioNamespace)
	if err != nil {
		klog.Infof("Cannot apply EF")
		return err
	}

	return nil

}

func BytesToString(data []byte) string {
	return string(data[:])
}

func retrieveJsonActions(rateLimitInstance networkingv1alpha1.RateLimit, baseName string) []byte {

	var output []byte

	actionsOutput := networkingv1alpha1.OutputRatelimitsEnvoyFilter{}

	//var Actions []networkingv1alpha1.Actions

	actionsOutput.RateLimits = make([]networkingv1alpha1.RateLimits, len(rateLimitInstance.Spec.Rate))

	//actions := make([]networkingv1alpha1.Actions, len(rateLimitInstance.Spec.Rate))

	//dimensionarray := make([]networkingv1alpha1.Dimensions, len(rateLimitInstance.Spec.Rate))

	for k, dimension := range rateLimitInstance.Spec.Rate {

		keyName := dimension.Dimensions[0].RequestHeader.DescriptorKey + "_" + dimension.Unit

		if dimension.Dimensions[0].RequestHeader.DescriptorKey == "" {
			actions := []networkingv1alpha1.Actions{
				{
					HeaderValueMatch: &networkingv1alpha1.HeaderValueMatch{
						DescriptorValue: dimension.Dimensions[len(dimension.Dimensions)-1].HeaderValueMatch.DescriptorValue,
						Headers: []networkingv1alpha1.Headers{
							{
								Name:        dimension.Dimensions[len(dimension.Dimensions)-1].HeaderValueMatch.Headers[0].Name,
								PrefixMatch: dimension.Dimensions[len(dimension.Dimensions)-1].HeaderValueMatch.Headers[0].PrefixMatch,
							},
						},
					},
				},
				{
					DestinationCluster: &networkingv1alpha1.DestinationClusterHeader{},
				},
			}
			actionsOutput.RateLimits[k].Actions = actions

		} else {
			actions := []networkingv1alpha1.Actions{

				{
					RequestHeaders: &networkingv1alpha1.RequestHeaders{
						DescriptorKey: keyName,
						HeaderName:    dimension.Dimensions[len(dimension.Dimensions)-1].RequestHeader.HeaderName,
					},
				},
				{
					DestinationCluster: &networkingv1alpha1.DestinationClusterHeader{},
				},
			}

			actionsOutput.RateLimits[k].Actions = actions
		}

	}

	/*	for _, dimension := range actionsOutput.RateLimitsActions {
		Actions = append(Actions, dimension.Actions...)
	}*/

	output, _ = json.Marshal(actionsOutput)

	return output

}

func createHttpFilterPatchValue(domain string, nameCluster string) (string, error) {
	//97:	payload = []byte(fmt.Sprintf(`{"typed_config":{"@type":"type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit","domain":"%s","rate_limit_service":{"transport_api_version": "V3","grpc_service":{"envoy_grpc":{"cluster_name":"%s"},"timeout":"1.25s"}}},"name":"envoy.filters.http.ratelimit"}`, domain, nameCluster))

	values := ratelimitTypes.HttpFilterPatchValues{
		Name: "envoy.filters.http.ratelimit",
		TypedConfig: ratelimitTypes.TypedConfig{
			Type:   "type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit",
			Domain: domain,
			RateLimitService: ratelimitTypes.RateLimitService{
				TransportAPIVersion: "V3",
				GRPCService: ratelimitTypes.GRPCService{
					Timeout: "1.25s",
					EnvoyGRPC: ratelimitTypes.EnvoyGRPC{
						ClusterName: nameCluster,
					},
				},
			},
		},
	}

	bytes, err := yaml.Marshal(&values)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func createClusterPatchValue(fqdn string, nameCluster string) (string, error) {
	//	payload := []byte(fmt.Sprintf(`{"connect_timeout":"1.25s","load_assignment":{"cluster_name":"%s","endpoints":[{"lb_endpoints":[{"endpoint":{"address":{"socket_address":{"address":"%s","port_value":8081}}}}]}]},"http2_protocol_options":{},"lb_policy":"ROUND_ROBIN","name":"%s","type":"STRICT_DNS"}`, fqdn, fqdn, nameCluster))

	values := ratelimitTypes.ClusterPatchValues{
		Name:                 nameCluster,
		Type:                 "STRICT_DNS",
		ConnectTimeout:       "1.25s",
		HTTP2ProtocolOptions: ratelimitTypes.HTTP2ProtocolOptions{},
		LbPolicy:             "ROUND_ROBIN",
		LoadAssignment: ratelimitTypes.LoadAssignment{
			ClusterName: fqdn,
			Endpoints: []ratelimitTypes.Endpoints{
				{
					LbEndpoints: []ratelimitTypes.LbEndpoints{
						{
							Endpoint: ratelimitTypes.Endpoint{
								Address: ratelimitTypes.Address{
									SocketAddress: ratelimitTypes.SocketAddress{
										Address:   fqdn,
										PortValue: 8081,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bytes, err := yaml.Marshal(&values)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
