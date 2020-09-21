package controllers

import (
	"fmt"

	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
)

func (r *RateLimitReconciler) applyEnvoyFilter(desired istio_v1alpha3.EnvoyFilter, found *istio_v1alpha3.EnvoyFilter, nameEnvoyFilter string) (ctrl.Result, error) {

	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: "istio-system",
		Name:      nameEnvoyFilter,
	}, found)
	if err != nil {
		fmt.Println(err)
		err = r.Create(context.TODO(), &desired)
		if err != nil {
			fmt.Println(err)
			return ctrl.Result{}, err
		}
	} else {

		applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("rate-limit-controller")}

		err = r.Patch(context.TODO(), &desired, client.Apply, applyOpts...)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil

}

func (r *RateLimitReconciler) deleteEnvoyFilter(envoyFilter istio_v1alpha3.EnvoyFilter) error {

	err := r.Delete(context.TODO(), &envoyFilter)
	if err != nil {
		fmt.Println("cannot delete envoy filter")
		return err
	}

	return nil

}

func (r *RateLimitReconciler) deleteConfigMap(configMapRateLimit v1.ConfigMap) error {

	err := r.Delete(context.TODO(), &configMapRateLimit)
	if err != nil {
		return err
	}

	return nil

}
