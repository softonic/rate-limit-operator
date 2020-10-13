package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
)

func (r *RateLimitReconciler) decomissionk8sObjectResources()  error {


	err := r.decomissionEnvoyFilters(r.EnvoyFilters)
	if err != nil {
		klog.Infof("Cannot remove EFs %v. Error %v",r.EnvoyFilters , err)
	}

	err = r.decomissionConfigMapRatelimit(r.configMapRateLimit)
	if err != nil {
		klog.Infof("Cannot remove EFs %v. Error %v",r.configMapRateLimit , err)
	}


	return nil
}


func (r *RateLimitReconciler) decomissionEnvoyFilters(EnvoyFilters []*istio_v1alpha3.EnvoyFilter )  error {


	for _,envoyfilter := range EnvoyFilters {
		err := r.deleteEnvoyFilter(*envoyfilter)
		if err != nil {
			return err
			klog.Infof("Cannot remove EF %v. Error %v",envoyfilter , err)
		}
	}

	return nil

}



func (r *RateLimitReconciler) decomissionConfigMapRatelimit(configMapRateLimit v1.ConfigMap )  error {

	err := r.deleteConfigMap(configMapRateLimit)
	if err != nil {
		return err
		klog.Infof("Cannot remove ConfigMap %v. Error %v",configMapRateLimit , err)
	}

	return nil

}


func ( r *RateLimitReconciler) decomissionDeploymentVolumes(sources []v1.VolumeProjection, volumes []v1.Volume) error {

	err := r.removeVolumeFromDeployment(r.DeploymentRL, sources, volumes)
	if err != nil {
		klog.Infof("Cannot remove VolumeSource from deploy %v. Error %v", r.DeploymentRL, err)
		return err
	}

	err = r.Update(context.TODO(), r.DeploymentRL)
	if err != nil {
		klog.Infof("Cannot Update Deployment %s. Error %v", "istio-system-ratelimit", err)
		return err
	}

}

func (r *RateLimitReconciler) applyEnvoyFilter(desired istio_v1alpha3.EnvoyFilter, found *istio_v1alpha3.EnvoyFilter, nameEnvoyFilter string, controllerNamespace string) (error) {

	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: controllerNamespace,
		Name:      nameEnvoyFilter,
	}, found)
	if err != nil {
		klog.Infof("Cannot Found EnvoyFilter %s. Error %v", found.Name, err)
		err = r.Create(context.TODO(), &desired)
		if err != nil {
			klog.Infof("Cannot Create EnvoyFilter %s. Error %v", desired.Name, err)
			return err
		}
	} else {

		applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("rate-limit-controller")}

		err = r.Patch(context.TODO(), &desired, client.Apply, applyOpts...)
		if err != nil {
			klog.Infof("Cannot Patch EnvoyFilter %s. Error %v", desired.Name, err)
			return err
		}
		return nil
	}

	return nil

}

func (r *RateLimitReconciler) deleteEnvoyFilter(envoyFilter istio_v1alpha3.EnvoyFilter) error {

	err := r.Delete(context.TODO(), &envoyFilter)
	if err != nil {
		klog.Infof("Cannot delete EnvoyFilter %s. Error %v", envoyFilter.Name, err)
		return err
	}

	return nil

}

func (r *RateLimitReconciler) CreateOrUpdateConfigMap(rateLimitInstance *networkingv1alpha1.RateLimit, controllerNamespace string, baseName string) error {

	configmapDesired, err := r.generateConfigMap(rateLimitInstance, controllerNamespace, baseName)
	if err != nil {
		klog.Infof("Cannot create %v, Error: %v", configmapDesired, err)
		return err
	}

	found := v1.ConfigMap{}

	r.configMapRateLimit, err = r.getConfigMap(baseName, controllerNamespace)
	if err != nil {
		err = r.Create(context.TODO(), &configmapDesired)
		if err != nil {
			//return ctrl.Result{}, client.IgnoreNotFound(err)
			klog.Infof("Cannot create %v, Error: %v", configmapDesired, err)
		}
	} else if !reflect.DeepEqual(configmapDesired, found) {

		applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("rate-limit-controller")}

		err = r.Patch(context.TODO(), &configmapDesired, client.Apply, applyOpts...)
		if err != nil {
			return err
		}
	}

	return nil

}

func (r *RateLimitReconciler) generateConfigMap(rateLimitInstance *networkingv1alpha1.RateLimit, desiredNamespace string, name string) (v1.ConfigMap, error) {

	configMapData := make(map[string]string)

	var output []byte

	descriptorOutput := networkingv1alpha1.DescriptorsParent{}

	descriptorOutput.Parent = make([]networkingv1alpha1.Dimensions, len(rateLimitInstance.Spec.Dimensions))

	descriptorOutput.Domain = name

	//dimensionOutput = []networkingv1alpha1.Dimensions{}

	for k, dimension := range rateLimitInstance.Spec.Dimensions {
		descriptorOutput.Parent[k].Key = dimension.Key
		descriptorOutput.Parent[k].Descriptors = append(descriptorOutput.Parent[k].Descriptors, dimension.Descriptors...)
		descriptorOutput.Parent[k].Actions = nil
	}

	output, _ = json.Marshal(descriptorOutput)

	y, _ := yaml.JSONToYAML(output)

	fileName := name + ".yaml"

	configMapData[fileName] = string(y)

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

func (r *RateLimitReconciler) UpdateDeployment(sources []v1.VolumeProjection, volumes []v1.Volume) error {

	err := r.addVolumeFromDeployment(r.DeploymentRL, sources, volumes)
	if err != nil {
		klog.Infof("Cannot add VolumeSource from deploy %v. Error %v", r.DeploymentRL, err)
		return err
	}

	err = r.Update(context.TODO(), r.DeploymentRL)
	if err != nil {
		klog.Infof("Cannot Update Deployment %s. Error %v", "istio-system-ratelimit", err)
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
