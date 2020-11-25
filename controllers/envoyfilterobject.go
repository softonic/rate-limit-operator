package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	"k8s.io/klog"
	"os"

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
}

func (r *RateLimitReconciler) prepareUpdateEnvoyFilterObjects(rateLimitInstance networkingv1alpha1.RateLimit, baseName string, controllerNamespace string) error {


	istioNamespace := os.Getenv("ISTIO_NAMESPACE")


	jsonActions := retrieveJsonActions(rateLimitInstance, baseName)

	namespace := rateLimitInstance.Spec.TargetRef.Namespace
	nameVirtualService := rateLimitInstance.Spec.TargetRef.Name

	virtualService, err := r.getVirtualService(namespace, nameVirtualService)
	if err != nil {
		klog.Infof("Virtualservice does not exists")
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

	// address := os.Getenv("ADDRESS_RATELIMIT_ENDPOINT")

	address := "istio-system-ratelimit"

	fqdn := address + "." + controllerNamespace + ".svc.cluster.local"

	payload := []byte(fmt.Sprintf(`{"connect_timeout": "1.25s", "hosts": [ { "socket_address": { "address": "%s", "port_value": 8081 } } ], "http2_protocol_options": {}, "lb_policy": "ROUND_ROBIN", "name": "rate_limit_service", "type": "STRICT_DNS" }`, fqdn))

	rawConfigCluster := json.RawMessage(payload)

	labels := make(map[string]string)

	labels = gatewaySelector

	envoyFilterObjectCluster := EnvoyFilterObject{
		Operation:             "ADD",
		ApplyTo:               "CLUSTER",
		RawConfig:             rawConfigCluster,
		TypeConfigObjectMatch: "Cluster",
		ClusterEndpoint:       address,
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

	payload = []byte(fmt.Sprintf(`{"config":{"domain":"%s","rate_limit_service":{"grpc_service":{"envoy_grpc":{"cluster_name":"rate_limit_service"},"timeout":"1.25s"}}},"name":"envoy.rate_limit"}`, domain))

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

	initJson := []byte(`{"route":{"rate_limits":[{"actions":`)
	finalJson := []byte(`}]}}`)

	intermediateJson := append(initJson, jsonActions...)

	rawConfigHTTPRoute := json.RawMessage(append(intermediateJson, finalJson...))

	//rawConfigHTTPRoute := json.RawMessage(`{"route":{"rate_limits":[{"actions":[{"request_headers":{"descriptor_key":"remote_address","header_name":"x-custom-user-ip"}},{"destination_cluster":{}}]}]}}`)

	envoyFilterObjectRouteConfiguration := EnvoyFilterObject{
		Operation:             "MERGE",
		ApplyTo:               "HTTP_ROUTE",
		RawConfig:             rawConfigHTTPRoute,
		TypeConfigObjectMatch: "RouteConfiguration",
		Context:               "GATEWAY",
		Labels:                labels,
		NameVhost:             nameVhost,
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

func retrieveJsonActions(rateLimitInstance networkingv1alpha1.RateLimit, baseName string) []byte {

	var output []byte

	actionsOutput := networkingv1alpha1.OutputRatelimitsEnvoyFilter{}

	var Actions []networkingv1alpha1.Actions

	actionsOutput.RateLimits = make([]networkingv1alpha1.ActionsEnvoyFilter, len(rateLimitInstance.Spec.Rate))

	actions := make([]networkingv1alpha1.Actions, len(rateLimitInstance.Spec.Rate))

	//dimensionarray := make([]networkingv1alpha1.Dimensions, len(rateLimitInstance.Spec.Rate))

	for k, dimension := range rateLimitInstance.Spec.Rate {

		actions = []networkingv1alpha1.Actions{
			{
				RequestHeaders: networkingv1alpha1.RequestHeaders{
					DescriptorKey: dimension.Unit,
					HeaderName:    dimension.Dimensions[len(dimension.Dimensions)-1].RequestHeader.HeaderName,
				},
			},
		}

		actionsOutput.RateLimits[k].Actions = actions

	}

	for _, dimension := range actionsOutput.RateLimits {
		Actions = append(Actions, dimension.Actions...)
	}

	output, _ = json.Marshal(Actions)

	return output

}
