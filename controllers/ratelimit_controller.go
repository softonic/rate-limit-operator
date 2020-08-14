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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"

	"fmt"

	"reflect"

	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	istio "istio.io/api/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	_ "log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RateLimitReconciler reconciles a RateLimit object
type RateLimitReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.softonic.io,resources=ratelimits,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=networking.softonic.io,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.softonic.io,resources=ratelimits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=envoyfilters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=envoyfilters/status,verbs=get

// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *RateLimitReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("ratelimit", req.NamespacedName)

	rateLimitInstance := &networkingv1alpha1.RateLimit{}

	err := r.Get(context.TODO(), req.NamespacedName, rateLimitInstance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// this hack is not working. Meanwhile I am using a real
	// VirtualService.networking.softonic.io deployed in the same
	// namespace ( config/samples/networking_v1alpha1_virtualservice.yaml )

	virtualService := &networkingv1alpha1.VirtualService{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Namespace: rateLimitInstance.Spec.TargetRef.Namespace,
		Name:      rateLimitInstance.Spec.TargetRef.Name,
	}, virtualService)
	if err != nil {
		fmt.Println("virtualservice not found")
		return ctrl.Result{}, err
	}

	// ensure rate limit envoy cluster (envoyfilter is created): deploy through manifest or control it by controller?

	config := `{"value":{"connect_timeout":"1.25s","hosts":[{"socket_address":null,"address":"local","port_value":8081}],"http2_protocol_options":{},"lb_policy":"ROUND_ROBIN","name":"rate_limit_service","type":"STRICT_DNS"}}`

	EnvoyConfigObjectPatch := buildClusterPatches(config)

	envoyFilter := istio.EnvoyFilter{
		WorkloadSelector: &istio.WorkloadSelector{
			Labels: map[string]string{"app": "istio-ingressgateway"},
		},
		ConfigPatches: EnvoyConfigObjectPatch,
	}

	fmt.Println(envoyFilter)

	/* 	spec:
	   	configPatches:
	   	- applyTo: CLUSTER
	   	  match:
	   		cluster:
	   		  service: istio-system-ratelimit.istio-system.svc.cluster.local
	   	  patch:
	   		operation: ADD
	   		value:
	   		  connect_timeout: 1.25s
	   		  hosts:
	   		  - socket_address:
	   			  address: istio-system-ratelimit.istio-system.svc.cluster.local
	   			  port_value: 8081
	   		  http2_protocol_options: {}
	   		  lb_policy: ROUND_ROBIN
	   		  name: rate_limit_service
	   		  type: STRICT_DNS
	   	workloadSelector:
	   	  labels:
	   		app: istio-ingressgateway */

	// istio namespace: Force application to be in the same namespace as istio? Pass istio root namespace as parameter?
	// Fill envoy filter and create/update it

	// Generate rate limit server configmap
	// create and update it

	controllerNamespace := "istio-system"

	configmapDesired, err := r.desiredConfigMap(rateLimitInstance, controllerNamespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	fmt.Println("we have a configmap to apply", configmapDesired)

	found := v1.ConfigMap{}

	err = r.Get(context.TODO(), types.NamespacedName{Namespace: controllerNamespace, Name: "test"}, &found)
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
