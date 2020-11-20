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
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/go-logr/logr"

	_ "log"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/klog"
	// "fmt"
)

// RateLimitReconciler reconciles a RateLimit object
type RateLimitReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	K8sObject
}

type K8sObject struct {
	EnvoyFilters       []*istio_v1alpha3.EnvoyFilter
	DeploymentRL       appsv1.Deployment
	configMapRateLimit v1.ConfigMap
}

// +kubebuilder:rbac:groups=networking.softonic.io,resources=ratelimits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=*,resources=virtualservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.softonic.io,resources=ratelimits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=envoyfilters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=envoyfilters/status,verbs=get
// +kubebuilder:rbac:groups=*,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *RateLimitReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("ratelimit", req.NamespacedName)

	rateLimitInstance := &networkingv1alpha1.RateLimit{}

	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, rateLimitInstance)
	if err != nil {
		klog.Infof("Cannot get Ratelimit CR %s. Error %v", rateLimitInstance.Name, err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// INIT VARIABLES

	baseName := req.Name

	// controllerNamespace := os.Getenv("CONTROLLER_NAMESPACE")
	// istioNamespace := os.Getenv("ISTIO_NAMESPACE")

	istioNamespace := "istio-system"

	controllerNamespace := "rate-limit-operator-system"

	nameVolume := "commonconfig-volume"

	finalizer := "ratelimit.networking.softonic.io"

	// INIT RESOURCES

	r.getK8sResources(baseName, istioNamespace, istioNamespace)

	volumes := constructVolumes(nameVolume, baseName)

	volumeProjectedSources := constructVolumeSources(baseName)

	// DECOMMISSION

	beingDeleted := rateLimitInstance.GetDeletionTimestamp() != nil

	if beingDeleted {

		if containsString(rateLimitInstance.GetFinalizers(), finalizer) {

			err = r.decomissionk8sObjectResources(baseName, controllerNamespace, istioNamespace)

			err = r.decomissionDeploymentVolumes(volumeProjectedSources, volumes)

			rateLimitInstance.SetFinalizers(remove(rateLimitInstance.GetFinalizers(), finalizer))
			err = r.Update(context.TODO(), rateLimitInstance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// prepare Envoy Filters and apply the needed changes
	err = r.prepareUpdateEnvoyFilterObjects(*rateLimitInstance, baseName, controllerNamespace)

	// Create ConfigMap Ratelimit
	err = r.CreateOrUpdateConfigMap(rateLimitInstance, istioNamespace, baseName)

	// Update deployment with ConfigMap values

	err = r.UpdateDeployment(volumeProjectedSources, volumes)

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
