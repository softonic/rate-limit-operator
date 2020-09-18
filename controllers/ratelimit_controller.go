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

package controllers

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/types"

	"fmt"

	"reflect"

	"strings"

	_ "log"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
	"github.com/softonic/rate-limit-operator/api/istio_v1beta1"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RateLimitReconciler reconciles a RateLimit object
type RateLimitReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

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

// +kubebuilder:rbac:groups=networking.softonic.io,resources=ratelimits,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=networking.softonic.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.softonic.io,resources=ratelimits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=envoyfilters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=envoyfilters/status,verbs=get

// +kubebuilder:rbac:groups=*,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *RateLimitReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("ratelimit", req.NamespacedName)

	rateLimitInstance := &networkingv1alpha1.RateLimit{}

	err := r.Get(context.TODO(), req.NamespacedName, rateLimitInstance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	baseName := req.Name

	controllerNamespace := "istio-system"

	finalizer := "ratelimit.networking.softonic.io"

	beingDeleted := rateLimitInstance.GetDeletionTimestamp() != nil

	envoyFilterCluster := r.getEnvoyFilter(baseName + "-cluster", controllerNamespace)

	envoyFilterHTTPFilter := r.getEnvoyFilter(baseName + "-envoy-filter", controllerNamespace)

	envoyFilterHTTPRoute := r.getEnvoyFilter(baseName + "-route", controllerNamespace)

	configMapRateLimit,err := r.getConfigMap(baseName, controllerNamespace)

	if beingDeleted {

		if containsString(rateLimitInstance.GetFinalizers(), finalizer) {

			err := r.deleteEnvoyFilter(*envoyFilterCluster)
			if err != nil {
				fmt.Println("cannot delete envoy filter")
				return ctrl.Result{}, err
			}

			err = r.deleteEnvoyFilter(*envoyFilterHTTPFilter)
			if err != nil {
				fmt.Println("cannot delete envoy filter")
				return ctrl.Result{}, err
			}

			err = r.deleteEnvoyFilter(*envoyFilterHTTPRoute)
			if err != nil {
				fmt.Println("cannot delete envoy filter")
				return ctrl.Result{}, err
			}

			err = r.deleteConfigMap(configMapRateLimit)
			if err != nil {
				fmt.Println("cannot delete config Map")
				return ctrl.Result{}, err
			}

			rateLimitInstance.SetFinalizers(remove(rateLimitInstance.GetFinalizers(), finalizer))
			err = r.Update(context.TODO(), rateLimitInstance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	virtualService := &istio_v1beta1.VirtualService{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Namespace: "ratelimitoperatortest",
		Name:      "vs-test",
	}, virtualService)
	if err != nil {
		fmt.Println(err)
		return ctrl.Result{}, err
	}

	firstElementHosts := strings.Join(virtualService.Spec.Hosts, "")

	nameVhost := firstElementHosts + ":80"

	address := "istio-system-ratelimit.istio-system.svc.cluster.local"

	payload := []byte(fmt.Sprintf(`{"connect_timeout": "1.25s", "hosts": [ { "socket_address": { "address": "%s", "port_value": 8081 } } ], "http2_protocol_options": {}, "lb_policy": "ROUND_ROBIN", "name": "rate_limit_service", "type": "STRICT_DNS" }`, address))

	rawConfigCluster := json.RawMessage(payload)

	labels := make(map[string]string)

	labels = rateLimitInstance.Spec.WorkloadSelector

	envoyFilterObjectCluster := EnvoyFilterObject{
		Operation:             "ADD",
		ApplyTo:               "CLUSTER",
		RawConfig:             rawConfigCluster,
		TypeConfigObjectMatch: "Cluster",
		ClusterEndpoint:       "istio-system-ratelimit.istio-system.svc.cluster.local",
		Labels:                labels,
	}


	envoyFilterClusterDesired := envoyFilterObjectCluster.composeEnvoyFilter(baseName + "-cluster", controllerNamespace)

	envoyFilterCluster = &istio_v1alpha3.EnvoyFilter{}

	result, err := r.applyEnvoyFilter(envoyFilterClusterDesired, envoyFilterCluster, baseName + "-cluster")
	if err != nil {
		return result, err
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


	envoyFilterHTTPFilterDesired := envoyFilterObjectListener.composeEnvoyFilter(baseName + "-envoy-filter", controllerNamespace)

	envoyFilterHTTPFilter = &istio_v1alpha3.EnvoyFilter{}

	result, err = r.applyEnvoyFilter(envoyFilterHTTPFilterDesired, envoyFilterHTTPFilter, baseName + "-envoy-filter")
	if err != nil {
		return result, err
	}

	rawConfigHTTPRoute := json.RawMessage(`{"route":{"rate_limits":[{"actions":[{"request_headers":{"descriptor_key":"remote_address","header_name":"x-custom-user-ip"}},{"destination_cluster":{}}]}]}}`)

	envoyFilterObjectRouteConfiguration := EnvoyFilterObject{
		Operation:             "MERGE",
		ApplyTo:               "HTTP_ROUTE",
		RawConfig:             rawConfigHTTPRoute,
		TypeConfigObjectMatch: "RouteConfiguration",
		Context:               "GATEWAY",
		Labels:                labels,
		NameVhost:             nameVhost,
	}


	envoyFilterHTTPRouteDesired := envoyFilterObjectRouteConfiguration.composeEnvoyFilter(baseName + "-route", controllerNamespace)

	envoyFilterHTTPRoute = &istio_v1alpha3.EnvoyFilter{}

	result, err = r.applyEnvoyFilter(envoyFilterHTTPRouteDesired, envoyFilterHTTPRoute, baseName + "-route")
	if err != nil {
		return result, err
	}


	configmapDesired, err := r.desiredConfigMap(rateLimitInstance, controllerNamespace, baseName)
	if err != nil {
		return ctrl.Result{}, err
	}

	found := v1.ConfigMap{}


	configMapRateLimit,err = r.getConfigMap(baseName, controllerNamespace)

    if err != nil {
		fmt.Println("could not find the configmap")

		err = r.Create(context.TODO(), &configmapDesired)
		if err != nil {
			fmt.Println("could not create the configmap")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		} else if err != nil {
			return ctrl.Result{}, err
		}
	} else if !reflect.DeepEqual(configmapDesired, found) {
		fmt.Println("ConfigMap exists but there are not the same")

		applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("rate-limit-controller")}

		err = r.Patch(context.TODO(), &configmapDesired, client.Apply, applyOpts...)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// patch server to restart if changes (CRC of above configs?)

	// read request
	// if delete, delete envoyfilter, config (and apply CRC to ratelimit server deploy)

	// if not delete
	// read CR's values
	// update envoyfilter, config (and apply CRC to ratelimit server deploy)

	return ctrl.Result{}, nil

}

func (r *RateLimitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.RateLimit{}).
		Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func remove(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
