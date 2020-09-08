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
	// "encoding/json"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/types"

	//"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	//"k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	"k8s.io/client-go/dynamic"
	// "k8s.io/client-go/rest"
	"k8s.io/client-go/discovery"
	// "k8s.io/client-go/discovery/cached/memory"
	// "k8s.io/client-go/restmapper"

	"fmt"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "k8s.io/apimachinery/pkg/api/meta"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"reflect"

	_ "log"

	"github.com/softonic/rate-limit-operator/api/istio_v1beta1"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	istio "istio.io/api/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RateLimitReconciler reconciles a RateLimit object
type RateLimitReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	ClientDynamic   dynamic.Interface
	DiscoveryClient *discovery.DiscoveryClient
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
		return ctrl.Result{}, err
	}

	// this hack is not working. Meanwhile I am using a real
	// VirtualService.networking.softonic.io deployed in the same
	// namespace ( config/samples/networking_v1alpha1_virtualservice.yaml )

	virtualService := &istio_v1beta1.VirtualService{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Namespace: "ratelimitoperatortest",
		Name:      "vs-test",
	}, virtualService)
	if err != nil {
		fmt.Println(err)
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

	//err = r.Create(context.TODO(), &envoyFilter)

	fmt.Println(envoyFilter)

	resourceScheme := schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1alpha3", Resource: "envoyfilters"}

	resp, err := r.ClientDynamic.Resource(resourceScheme).Namespace("istio-system").Get(context.TODO(), "ratelimit-cluster", v12.GetOptions{})
	if err != nil {
		fmt.Println("Error getting Envoy Filter")
	}

	// here we will create the envoyfilter of type unstructured.Unstructured

	fmt.Println(resp)

	const deploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
`

	const envoyManifest = `
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: ratelimit-cluster
  namespace: istio-system
spec:
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
      app: istio-ingressgateway
	`

	// envoyFilterUnstructured := &unstructured.Unstructured{
	// 	Object: map[string]interface{}{
	// 		"apiVersion": "networking.istio.io/v1alpha3",
	// 		"kind":       "EnvoyFilter",
	// 		"metadata": map[string]interface{}{
	// 			"name": "ratelimit-cluster-dup",
	// 			"namespace": "istio-system",
	// 		},
	// 		"spec": map[string]interface{}{
	// 			"template": map[string]interface{}{
	// 				"metadata": map[string]interface{}{
	// 					"labels": map[string]interface{}{
	// 						"app": "demo",
	// 					},
	// 				},

	// 				"spec": map[string]interface{}{
	// 					"containers": []map[string]interface{}{
	// 						{
	// 							"name":  "web",
	// 							"image": "nginx:1.12",
	// 							"ports": []map[string]interface{}{
	// 								{
	// 									"name":          "http",
	// 									"protocol":      "TCP",
	// 									"containerPort": 80,
	// 								},
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 	},
	// }

	// var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	// // 1. Prepare a RESTMapper to find GVR

	// mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(r.DiscoveryClient))

	// obj := &unstructured.Unstructured{}
	// rto, gvk, err := decUnstructured.Decode([]byte(deploymentYAML), nil, obj)
	// if err != nil {
	// 	fmt.Println("Decode failed")
	// }

	// fmt.Println(rto)

	// // 4. Find GVR
	// mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	// if err != nil {
	// 	fmt.Println("mapping failed")
	// }

	// // 5. Obtain REST interface for the GVR
	// var dr dynamic.ResourceInterface

	// if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
	// 	// namespaced resources should specify the namespace
	// 	dr = r.ClientDynamic.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	// } else {
	// 	// for cluster-wide resources
	// 	dr = r.ClientDynamic.Resource(mapping.Resource)
	// }

	// data, err := json.Marshal(obj)
	// if err != nil {
	// 	fmt.Println("Marshal failed")
	// }

	// _, err = dr.Patch(context.TODO(), obj.GetName(), types.ApplyPatchType, data, v12.PatchOptions{
	// 	FieldManager: "sample-controller",
	// })

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
