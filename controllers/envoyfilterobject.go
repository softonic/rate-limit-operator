package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
	"github.com/softonic/rate-limit-operator/api/istio_v1beta1"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
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
}

func (r *RateLimitReconciler) prepareUpdateEnvoyFilterObjects(rateLimitInstance networkingv1alpha1.RateLimit, baseName string, controllerNamespace string) error {

	// controllerNamespace := os.Getenv("ISTIO_NAMESPACE")

	istioNamespace := "istio-system"

	jsonActions := retrieveJsonActions(rateLimitInstance, baseName)

	namespace := rateLimitInstance.Spec.TargetRef.Namespace
	nameVirtualService := rateLimitInstance.Spec.TargetRef.Name

	virtualService := &istio_v1beta1.VirtualService{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      nameVirtualService,
	}, virtualService)
	if err != nil {
		klog.Infof("Virtualservice does not exists")
	}

	firstElementHosts := strings.Join(virtualService.Spec.Hosts, "")

	nameVhost := firstElementHosts + ":80"

	// address := os.Getenv("ADDRESS_RATELIMIT_ENDPOINT")

	address := "istio-system-ratelimit"

	fqdn := address + "." + controllerNamespace + ".svc.cluster.local"

	payload := []byte(fmt.Sprintf(`{"connect_timeout": "1.25s", "hosts": [ { "socket_address": { "address": "%s", "port_value": 8081 } } ], "http2_protocol_options": {}, "lb_policy": "ROUND_ROBIN", "name": "rate_limit_service", "type": "STRICT_DNS" }`, fqdn))

	rawConfigCluster := json.RawMessage(payload)

	labels := make(map[string]string)

	labels = rateLimitInstance.Spec.WorkloadSelector

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

	var Actions []networkingv1alpha1.Actions

	var Dimensions []networkingv1alpha1.Dimensions

	Dimensions = make([]networkingv1alpha1.Dimensions, len(rateLimitInstance.Spec.Dimensions))


	for k, dimension := range rateLimitInstance.Spec.Dimensions {
		//Dimensions = append(Dimensions, dimension)
		Dimensions[k].Actions = append(Dimensions[k].Actions, dimension.Actions[0])
		Dimensions[k].Key = ""
		Dimensions[k].Descriptors = nil
	}


	for _, dimension := range Dimensions {
		Actions = append(Actions, dimension.Actions...)
	}

	output, _ = json.Marshal(Actions)

	return output

}