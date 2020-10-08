package controllers

import (
	"fmt"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
)

func (r *RateLimitReconciler) applyEnvoyFilter(desired istio_v1alpha3.EnvoyFilter, found *istio_v1alpha3.EnvoyFilter, nameEnvoyFilter string, controllerNamespace string) (ctrl.Result, error) {

	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: controllerNamespace,
		Name:      nameEnvoyFilter,
	}, found)
	if err != nil {
		klog.Infof("Cannot Found EnvoyFilter %s. Error %v", found.Name, err)
		err = r.Create(context.TODO(), &desired)
		if err != nil {
			klog.Infof("Cannot Create EnvoyFilter %s. Error %v", desired.Name, err)
			return ctrl.Result{}, err
		}
	} else {

		applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("rate-limit-controller")}

		err = r.Patch(context.TODO(), &desired, client.Apply, applyOpts...)
		if err != nil {
			klog.Infof("Cannot Patch EnvoyFilter %s. Error %v", desired.Name, err)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil

}

func (r *RateLimitReconciler) deleteEnvoyFilter(envoyFilter istio_v1alpha3.EnvoyFilter) error {

	err := r.Delete(context.TODO(), &envoyFilter)
	if err != nil {
		klog.Infof("Cannot delete EnvoyFilter %s. Error %v", envoyFilter.Name, err)
		return err
	}

	return nil

}

func (r *RateLimitReconciler) createDesiredConfigMap(rateLimitInstance *networkingv1alpha1.RateLimit, desiredNamespace string, name string) (v1.ConfigMap, error) {

	configMapData := make(map[string]string)

	configyaml := ConfigMaptoYAML{}

	for _, dimension := range rateLimitInstance.Spec.Dimensions {
		configyaml = ConfigMaptoYAML{
			DescriptorsParent: []DescriptorsParent{
				{
					Key: dimension.Key,
					Descriptors: []Descriptors{
						{
							Value:      "",
							ratelimits: RateLimitS{},
						},
					},
				},
			},
			Domain: name,
		}
	}

	/*for _, dimension := range rateLimitInstance.Spec.Dimensions {
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
	}*/

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
		klog.Infof("Cannot delete ConfigMap %s. Error %v", configMapRateLimit.Name, err)
		return err
	}

	return nil

}

func (r *RateLimitReconciler) removeVolumeFromDeployment(deploy *appsv1.Deployment, sources []v1.VolumeProjection, volumes []v1.Volume) error {

	for _, v := range deploy.Spec.Template.Spec.Volumes {
		if v.Name == "commonconfig-volume" && len(v.VolumeSource.Projected.Sources) > 1 {
			i := 0
			for _, n := range v.VolumeSource.Projected.Sources {
				fmt.Println("this is the first element of the sources", n)
				for _, p := range sources {
					if n.ConfigMap.Name == p.ConfigMap.Name {
						fmt.Println("Match with an element")
					} else {
						v.VolumeSource.Projected.Sources[i] = n
						i++
					}
				}
			}
			v.VolumeSource.Projected.Sources = v.VolumeSource.Projected.Sources[:i]
		} else if v.Name == "commonconfig-volume" && len(v.VolumeSource.Projected.Sources) == 1 {
			fmt.Println("remove volumes and volumemounts", v.Name)
			deploy.Spec.Template.Spec.Volumes = nil
			deploy.Spec.Template.Spec.Containers[0].VolumeMounts = nil
		}
	}

	return nil

}

func (r *RateLimitReconciler) addVolumeFromDeployment(deploy *appsv1.Deployment, sources []v1.VolumeProjection, volumes []v1.Volume) error {

	defaultVolumeMount := []v1.VolumeMount{
		{
			Name:      "commonconfig-volume",
			MountPath: "/data/ratelimit/config",
		},
	}

	if len(deploy.Spec.Template.Spec.Volumes) == 0 {
		deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volumes...)
		deploy.Spec.Template.Spec.Containers[0].VolumeMounts = defaultVolumeMount
		return nil
	}

	count := 0
	for _, v := range deploy.Spec.Template.Spec.Volumes {
		if v.Name == "commonconfig-volume" {
			v.VolumeSource.Projected.Sources = append(v.VolumeSource.Projected.Sources, sources...)
		} else {
			count++
			//deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volumes...)
		}
	}

	if count > 0 {
		deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volumes...)
		deploy.Spec.Template.Spec.Containers[0].VolumeMounts = append(deploy.Spec.Template.Spec.Containers[0].VolumeMounts, defaultVolumeMount...)
	}

	return nil

}
