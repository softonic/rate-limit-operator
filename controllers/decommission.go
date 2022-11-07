package controllers

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	clientIstio "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func (r *RateLimitReconciler) decomissionk8sObjectResources(baseName string, controllerNamespace string, istioNamespace string) error {

	envoyFilterCluster := r.getEnvoyFilter(baseName+"-cluster", istioNamespace)

	err := r.deleteEnvoyFilter(envoyFilterCluster)
	if err != nil {
		return err
	}

	envoyFilterHTTPFilter := r.getEnvoyFilter(baseName+"-envoy-filter", istioNamespace)

	err = r.deleteEnvoyFilter(envoyFilterHTTPFilter)
	if err != nil {
		return err
	}

	envoyFilterHTTPRoute := r.getEnvoyFilter(baseName+"-route", istioNamespace)

	err = r.deleteEnvoyFilter(envoyFilterHTTPRoute)
	if err != nil {
		return err
	}

	err = r.decomissionConfigMapRatelimit(r.configMapRateLimit)
	if err != nil {
		klog.Infof("Cannot remove EFs %v. Error %v", r.configMapRateLimit, err)
	}

	return nil
}

func (r *RateLimitReconciler) deleteEnvoyFilter(envoyFilter *clientIstio.EnvoyFilter) error {

	err := r.Delete(context.TODO(), envoyFilter)
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

func (r *RateLimitReconciler) decomissionDeploymentVolumes(sources []v1.VolumeProjection, volumes []v1.Volume, controllerNamespace string, deploymentName string) error {

	var err error

	r.DeploymentRL, err = r.getDeployment(controllerNamespace, deploymentName)
	if err != nil {
		klog.Infof("Cannot Found Deployment %s. Error %v", deploymentName, err)
		return err
	} else {
		klog.Infof("This is the  Deployment %s found in decomission. Annotations: %v", deploymentName, r.DeploymentRL.Spec.Template.Annotations)
	}

	err = r.removeVolumeFromDeployment(sources, volumes)
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
		klog.Infof("Volumes already deployed in deployment are: %v", v)
		if v.Name == "commonconfig-volume" && len(v.VolumeSource.Projected.Sources) > 1 {
			i := 0
			klog.Info("Entering first for: there are more than 1 sources and the volume is the correct one")
			for _, n := range v.VolumeSource.Projected.Sources {
				klog.Infof("the source is: %v", n)
				for _, p := range sources {
					if n.ConfigMap.Name == p.ConfigMap.Name {
					} else {
						v.VolumeSource.Projected.Sources[i] = n
						i++
					}
				}
			}
			v.VolumeSource.Projected.Sources = v.VolumeSource.Projected.Sources[:i]
		}
		//else if v.Name == "commonconfig-volume" && len(v.VolumeSource.Projected.Sources) == 1 {
		//	r.DeploymentRL.Spec.Template.Spec.Volumes = nil
		//	r.DeploymentRL.Spec.Template.Spec.Containers[0].VolumeMounts = nil

		//}
	}

	return nil

}
