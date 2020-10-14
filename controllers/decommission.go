package controllers

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	"fmt"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
)

func (r *RateLimitReconciler) decomissionk8sObjectResources() error {

	err := r.decomissionEnvoyFilters(r.EnvoyFilters)
	if err != nil {
		klog.Infof("Cannot remove EFs %v. Error %v", r.EnvoyFilters, err)
	}

	err = r.decomissionConfigMapRatelimit(r.configMapRateLimit)
	if err != nil {
		klog.Infof("Cannot remove EFs %v. Error %v", r.configMapRateLimit, err)
	}

	return nil
}

func (r *RateLimitReconciler) decomissionEnvoyFilters(EnvoyFilters []*istio_v1alpha3.EnvoyFilter) error {

	for _, envoyfilter := range EnvoyFilters {
		err := r.deleteEnvoyFilter(*envoyfilter)
		if err != nil {
			klog.Infof("Cannot remove EF %v. Error %v", envoyfilter, err)
			return err
		}
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

func (r *RateLimitReconciler) decomissionConfigMapRatelimit(configMapRateLimit v1.ConfigMap) error {

	err := r.deleteConfigMap(configMapRateLimit)
	if err != nil {
		klog.Infof("Cannot remove ConfigMap %v. Error %v", configMapRateLimit, err)
		return err
	}

	return nil

}

func (r *RateLimitReconciler) deleteConfigMap(configMapRateLimit v1.ConfigMap) error {

	err := r.Delete(context.TODO(), &configMapRateLimit)
	if err != nil {
		klog.Infof("Cannot delete ConfigMap %s. Error %v", configMapRateLimit.Name, err)
		return err
	}

	return nil

}

func (r *RateLimitReconciler) decomissionDeploymentVolumes(sources []v1.VolumeProjection, volumes []v1.Volume) error {

	err := r.removeVolumeFromDeployment(sources, volumes)
	if err != nil {
		klog.Infof("Cannot remove VolumeSource from deploy %v. Error %v", r.DeploymentRL, err)
		return err
	}

	err = r.Update(context.TODO(), &r.DeploymentRL)
	if err != nil {
		klog.Infof("Cannot Update Deployment %s. Error %v", "istio-system-ratelimit", err)
		return err
	}

	return nil

}

func (r *RateLimitReconciler) removeVolumeFromDeployment(sources []v1.VolumeProjection, volumes []v1.Volume) error {

	for _, v := range r.DeploymentRL.Spec.Template.Spec.Volumes {
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
			//	r.DeploymentRL.Spec.Template.Spec.Volumes = nil
			//	r.DeploymentRL.Spec.Template.Spec.Containers[0].VolumeMounts = nil
		}
	}

	return nil

}
