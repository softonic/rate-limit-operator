package controllers

import (
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"

	"os"
)

func (r *RateLimitReconciler) applyEnvoyFilter(desired istio_v1alpha3.EnvoyFilter, found *istio_v1alpha3.EnvoyFilter, nameEnvoyFilter string) (ctrl.Result, error) {

	controllerNamespace := os.Getenv("ISTIO_NAMESPACE")


	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: controllerNamespace,
		Name:      nameEnvoyFilter,
	}, found)
	if err != nil {
		klog.Errorf("Cannot Found EnvoyFilter %s. Error %v", found.Name, err)
		err = r.Create(context.TODO(), &desired)
		if err != nil {
			klog.Errorf("Cannot Create EnvoyFilter %s. Error %v", desired.Name, err)
			return ctrl.Result{}, err
		}
	} else {

		applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("rate-limit-controller")}

		err = r.Patch(context.TODO(), &desired, client.Apply, applyOpts...)
		if err != nil {
			klog.Errorf("Cannot Patch EnvoyFilter %s. Error %v", desired.Name, err)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil

}

func (r *RateLimitReconciler) deleteEnvoyFilter(envoyFilter istio_v1alpha3.EnvoyFilter) error {

	err := r.Delete(context.TODO(), &envoyFilter)
	if err != nil {
		klog.Errorf("Cannot delete EnvoyFilter %s. Error %v",envoyFilter.Name , err)
		return err
	}

	return nil

}

func (r *RateLimitReconciler) createDesiredConfigMap(rateLimitInstance *networkingv1alpha1.RateLimit, desiredNamespace string, name string) (v1.ConfigMap, error) {

	configMapData := make(map[string]string)

	configyaml := ConfigMaptoYAML{}

	for _, dimension := range rateLimitInstance.Spec.Dimensions {
		// we assume the second dimension is always destination_cluster
		for _, ratelimitdimension := range dimension {
			for n, dimensionKey := range ratelimitdimension {
				if n == "descriptor_key" {
					configyaml = ConfigMaptoYAML{
						DescriptorsParent: []DescriptorsParent{
							{
								Descriptors: []Descriptors{
									{
										Key:   "destination_cluster",
										Value: rateLimitInstance.Spec.DestinationCluster,
										RateLimit: RateLimitDescriptor{
											RequestsPerUnit: rateLimitInstance.Spec.RequestsPerUnit,
											Unit:            rateLimitInstance.Spec.Unit,
										},
									},
								},
								Key: dimensionKey,
							},
						},
						Domain: name,
					}
				}
			}
		}
	}

	configYamlFile, _ := yaml.Marshal(&configyaml)

	fileName := name + ".yaml"

	configMapData[fileName] = string(configYamlFile)

	configMap := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: desiredNamespace,
		},
		Data: configMapData,
	}

	return configMap, nil

}

func (r *RateLimitReconciler) deleteConfigMap(configMapRateLimit v1.ConfigMap) error {

	err := r.Delete(context.TODO(), &configMapRateLimit)
	if err != nil {
		klog.Errorf("Cannot delete ConfigMap %s. Error %v",configMapRateLimit.Name , err)
		return err
	}

	return nil

}
