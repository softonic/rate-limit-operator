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
	_ "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	_ "log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
)

// RateLimitReconciler reconciles a RateLimit object
type RateLimitReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.softonic.io,resources=ratelimits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.softonic.io,resources=ratelimits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.istio.io,resources=envoyfilters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=envoyfilters/status,verbs=get

func (r *RateLimitReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("ratelimit", req.NamespacedName)

	rateLimitCR := &networkingv1alpha1.RateLimit{}

	err := r.Get(context.TODO(), req.NamespacedName, rateLimitCR)
	if err != nil {
		return ctrl.Result{}, err
	}

	/*	controllerNamespace := "rate-limit-controller"

		configmapDesired, err := r.desiredConfigDomainMap(rateLimitCR, controllerNamespace)
		if err != nil {
			return ctrl.Result{}, err
		}

		found := v1.ConfigMap{}

		err = r.Get(context.TODO(), types.NamespacedName{Namespace: controllerNamespace, Name: rateLimitCR.Spec.NameDomainConfig}, &found)
		if err != nil {
			err = r.Create(context.TODO(), &configmapDesired)
			if err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			} else if err != nil {
				return ctrl.Result{}, err
			}
		} else if !reflect.DeepEqual(configmapDesired, found) {
			applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("rate-limit-controller")}
			err = r.Patch(context.TODO(), &configmapDesired, client.Apply, applyOpts...)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	*/
	// if err := controllerutil.SetControllerReference(rateLimitCR, configmap, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	//   }

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
