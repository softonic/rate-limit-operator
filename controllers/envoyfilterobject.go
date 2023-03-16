package controllers

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	"k8s.io/klog"

	// "os"
	"strings"
)

type EnvoyFilterObject struct {
	ApplyTo               string
	Operation             string
	RawConfig             json.RawMessage
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
		return err
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

	payload := []byte(fmt.Sprintf(`{"connect_timeout":"1.25s","load_assignment":{"cluster_name":"%s","endpoints":[{"lb_endpoints":[{"endpoint":{"address":{"socket_address":{"address":"%s","port_value":8081}}}}]}]},"http2_protocol_options":{},"lb_policy":"ROUND_ROBIN","name":"%s","type":"STRICT_DNS"}`, fqdn, fqdn, nameCluster))

	rawConfigCluster := json.RawMessage(payload)

	labels := make(map[string]string)

	labels = gatewaySelector

	envoyFilterObjectCluster := EnvoyFilterObject{
		Operation:             "ADD",
		ApplyTo:               "CLUSTER",
		RawConfig:             rawConfigCluster,
		TypeConfigObjectMatch: "Cluster",
		ClusterEndpoint:       fqdn,
		Labels:                labels,
	}

	envoyFilterClusterDesired := envoyFilterObjectCluster.composeEnvoyFilter(baseName+"-cluster", istioNamespace)

	envoyFilterCluster := &istio_v1alpha3.EnvoyFilter{}

	err = r.applyEnvoyFilter(envoyFilterClusterDesired, envoyFilterCluster, baseName+"-cluster", istioNamespace)
	if err != nil {
		klog.Infof("Cannot apply EF")
		return err
	}

	domain := baseName

	payload = []byte(fmt.Sprintf(`{"typed_config":{"@type":"type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit","domain":"%s","rate_limit_service":{"transport_api_version": "V3","grpc_service":{"envoy_grpc":{"cluster_name":"%s"},"timeout":"1.25s"}}},"name":"envoy.filters.http.ratelimit"}`, domain, nameCluster))

	rawConfigHTTPFilter := json.RawMessage(payload)

	envoyFilterObjectListener := EnvoyFilterObject{
		Operation:             "INSERT_BEFORE",
		ApplyTo:               "HTTP_FILTER",
		RawConfig:             rawConfigHTTPFilter,
		TypeConfigObjectMatch: "Listener",
		Context:               "GATEWAY",
		Labels:                labels,
	}

	envoyFilterHTTPFilterDesired := envoyFilterObjectListener.composeEnvoyFilter(baseName+"-envoy-filter", istioNamespace)

	envoyFilterHTTPFilter := &istio_v1alpha3.EnvoyFilter{}

	err = r.applyEnvoyFilter(envoyFilterHTTPFilterDesired, envoyFilterHTTPFilter, baseName+"-envoy-filter", istioNamespace)
	if err != nil {
		klog.Infof("Cannot apply EF")
		return err
	}

	initJson := []byte(`{"route":`)
	finalJson := []byte(`}`)

	intermediateJson := append(initJson, jsonActions...)

	rawConfigHTTPRoute := json.RawMessage(append(intermediateJson, finalJson...))

	//rawConfigHTTPRoute := json.RawMessage(`{"route":{"rate_limits":[{"actions":[{"request_headers":{"descriptor_key":"remote_address","header_name":"x-custom-user-ip"}},{"destination_cluster":{}}]}]}}`)

	var routesToApply []string

	if len(rateLimitInstance.Spec.ApplyToRoutes) > 0 {
		routesToApply = rateLimitInstance.Spec.ApplyToRoutes
	}

	envoyFilterObjectRouteConfiguration := EnvoyFilterObject{
		Operation:             "MERGE",
		ApplyTo:               "HTTP_ROUTE",
		RawConfig:             rawConfigHTTPRoute,
		TypeConfigObjectMatch: "RouteConfiguration",
		Context:               "GATEWAY",
		Labels:                labels,
		NameVhost:             nameVhost,
		Routes:                routesToApply,
	}

	envoyFilterHTTPRouteDesired := envoyFilterObjectRouteConfiguration.composeEnvoyFilter(baseName+"-route", istioNamespace)

	envoyFilterHTTPRoute := &istio_v1alpha3.EnvoyFilter{}

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

	actions := make([]networkingv1alpha1.Actions, len(rateLimitInstance.Spec.Rate))

	//dimensionarray := make([]networkingv1alpha1.Dimensions, len(rateLimitInstance.Spec.Rate))

	for k, dimension := range rateLimitInstance.Spec.Rate {

		keyName := dimension.Dimensions[0].RequestHeader.DescriptorKey + "_" + dimension.Unit

		if dimension.Dimensions[0].RequestHeader.DescriptorKey == "" {
			actions = []networkingv1alpha1.Actions{
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
			actions = []networkingv1alpha1.Actions{

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
