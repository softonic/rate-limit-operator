package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ghodss/yaml"
	"strconv"
	"time"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
)

func (r *RateLimitReconciler) applyEnvoyFilter(desired istio_v1alpha3.EnvoyFilter, found *istio_v1alpha3.EnvoyFilter, nameEnvoyFilter string, controllerNamespace string) error {

	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: controllerNamespace,
		Name:      nameEnvoyFilter,
	}, found)
	if err != nil {
		klog.Infof("Cannot Found EnvoyFilter %s before creating. %v", found.Name, err)
		err = r.Create(context.TODO(), &desired)
		if err != nil {
			klog.Infof("Cannot Create EnvoyFilter %s. Error %v", desired.Name, err)
			return err
		}
		klog.Infof("Creating %s...", desired.Name)
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

func (r *RateLimitReconciler) CreateOrUpdateConfigMap(rateLimitInstance *networkingv1alpha1.RateLimit, controllerNamespace string, baseName string, deploymentName string) error {

	var err error

	cm, err := r.generateConfigMap(rateLimitInstance, controllerNamespace, baseName)
	if err != nil {
		klog.Infof("Cannot generate %v, Error: %v", cm, err)
		return err
	}

	found := v1.ConfigMap{}

	found, err = r.getConfigMap(baseName, controllerNamespace)
	if err != nil {
		err = r.Create(context.TODO(), &cm)
		if err != nil {
			//return ctrl.Result{}, client.IgnoreNotFound(err)
			klog.Infof("Cannot create %v, Error: %v", cm, err)
		}
	} else if !reflect.DeepEqual(cm, found) {

		applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("rate-limit-controller")}

		klog.Infof("the 2 resources are not the same, we should patch the deployment")

		err = r.Patch(context.TODO(), &cm, client.Apply, applyOpts...)
		if err != nil {
			klog.Infof("Cannot patch cm. Error: %v",  err)
			return err
		}

		r.mutex.Lock()
		defer r.mutex.Unlock()

/*		deploy := &appsv1.Deployment{}
		err := r.Get(context.TODO(), client.ObjectKey{
			Namespace: controllerNamespace,
			Name:      "rate-limit-operator-ratelimit",
		}, deploy)
		if err != nil {
			klog.Infof("Cannot Get Deployment %s. Error %v", "istio-system-ratelimit", err)
			return err
		}*/

		r.DeploymentRL, err = r.getDeployment(controllerNamespace, deploymentName)
		if err != nil {
			klog.Infof("Cannot Found Deployment %s. Error %v", deploymentName, err)
			return err
		} else {
			klog.Infof("This is the  Deployment %s found in the patch operation. Annotations: %v", deploymentName, r.DeploymentRL.Spec.Template.Annotations)
		}

		epoch := strconv.FormatInt(time.Now().Unix(), 10)

		r.DeploymentRL.Spec.Template.Annotations["date"] = epoch

		err = r.Update(context.TODO(), &r.DeploymentRL)
		if err != nil {
			klog.Infof("Cannot Update Deployment %s in patch operation. Error %v", r.DeploymentRL.Name, err)
			return err
		} else {
			klog.Infof("Deployment updated inside patch loop.")
		}
		

	}

	return nil

}

func (r *RateLimitReconciler) generateConfigMap(rateLimitInstance *networkingv1alpha1.RateLimit, controllerNamespace string, name string) (v1.ConfigMap, error) {

	configMapData := make(map[string]string)

	var err error

	var output []byte

	descriptorOutput := networkingv1alpha1.OutputConfig{}

	descriptorOutput.DescriptorsParent = make([]networkingv1alpha1.DescriptorsParent, len(rateLimitInstance.Spec.Rate))

	descriptorOutput.Domain = name

	// get Destination Cluster

	nameVirtualService := rateLimitInstance.Spec.TargetRef.Name

	var value string

	if rateLimitInstance.Spec.DestinationCluster != "" {
		value = rateLimitInstance.Spec.DestinationCluster
	} else {
		value, err = r.getDestinationClusterFromVirtualService("istio-system", nameVirtualService)
		if err != nil {
			klog.Infof("Cannot generate configmap as we cannot find a host destination cluster")
			return v1.ConfigMap{}, err
		}
	}

	for k, dimension := range rateLimitInstance.Spec.Rate {
		descriptorOutput.DescriptorsParent[k].Key = dimension.Dimensions[0].RequestHeader.DescriptorKey + "_" + dimension.Unit
		descriptor := networkingv1alpha1.Descriptors{
			Key: "destination_cluster",
			RateLimit: networkingv1alpha1.RateLimitPerDescriptor{
				RequestsPerUnit: dimension.RequestPerUnit,
				Unit:            dimension.Unit,
			},
			Value: value,
		}
		descriptorOutput.DescriptorsParent[k].Descriptors = append(descriptorOutput.DescriptorsParent[k].Descriptors, descriptor)
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
			Namespace: controllerNamespace,
		},
		Data: configMapData,
	}

	return configMap,nil

}

func (r *RateLimitReconciler) getDestinationClusterFromVirtualService(namespace string, nameVirtualService string) (string, error) {

	virtualService, err := r.getVirtualService(namespace, nameVirtualService)
	if err != nil {
		klog.Infof("Virtualservice does not exists")
		return "", err
	}

	subset := ""
	destination := ""

	for k,routes := range virtualService.Spec.Http {
		if routes.Route[k].Destination.Host != "" {
			destination = routes.Route[k].Destination.Host
			subset = routes.Route[k].Destination.Subset
		}
	}

	if destination == "" {
		return "", errors.New("cannot find any suitable destinationCluster")
	}

	//outbound|80|prod|server.digitaltrends-v1.svc.cluster.local


	destinationCluster := "outbound|80|" + subset + "|" + destination

	return destinationCluster, errors.New("cannot find any suitable destinationCluster")

}

func (r *RateLimitReconciler) UpdateDeployment(volumeProjectedSources []v1.VolumeProjection, volumes []v1.Volume, controllerNamespace string, deploymentName string) error {

	var err error

	r.mutex.Lock()
	defer r.mutex.Unlock()

	time.Sleep(4 * time.Second)

	r.DeploymentRL, err = r.getDeployment(controllerNamespace, deploymentName)
	if err != nil {
		klog.Infof("Cannot Found Deployment %s. Error %v", deploymentName, err)
		return err
	} else {
		klog.Infof("This is the  Deployment %s found later on last function. Annotations: %v", deploymentName, r.DeploymentRL.Spec.Template.Annotations)
	}

	err = r.addVolumeFromDeployment(volumeProjectedSources, volumes)
	if err != nil {
		klog.Infof("Cannot add VolumeSource from deploy %v. Error %v", r.DeploymentRL, err)
		return err
	}


	err = r.Update(context.TODO(), &r.DeploymentRL)
	if err != nil {
		err = r.Update(context.TODO(), &r.DeploymentRL)
		if err != nil {
			klog.Infof("Cannot Update Deployment %s. Error %v", r.DeploymentRL.Name, err)
			return err
		}
	}



	return nil

}

func (r *RateLimitReconciler) addVolumeFromDeployment(volumeProjectedSources []v1.VolumeProjection, volumes []v1.Volume) error {

	
	var volumeProjectedSourcesToApply []v1.VolumeProjection

	defaultVolumeMount := []v1.VolumeMount{
		{
			Name:      "commonconfig-volume",
			MountPath: "/data/ratelimit/config",
		},
	}

	//if len(r.DeploymentRL.Spec.Template.Spec.Volumes) == 0 {
	//	r.DeploymentRL.Spec.Template.Spec.Volumes = append(r.DeploymentRL.Spec.Template.Spec.Volumes, volumes...)
	//	r.DeploymentRL.Spec.Template.Spec.Containers[0].VolumeMounts = defaultVolumeMount
	//	return nil
	//}

	count := 0
	exists := true
	for _, v := range r.DeploymentRL.Spec.Template.Spec.Volumes {
		if v.Name == "commonconfig-volume" {
			for _,sourceToApply := range volumeProjectedSources {
				for _,sourceAlreadyExists := range v.VolumeSource.Projected.Sources {
					if sourceToApply.ConfigMap.Name == sourceAlreadyExists.ConfigMap.Name {
						// this configmap is already in the volume projected sources, not need to include
						exists = true
						break
					} else {
						// there is no coincidence
						exists = false
					}
				}
				// If the sourcetoApply does not exists in the already mounted sources, append to the slice volumeProjectedSourcesToApply
				if exists == false {
					volumeProjectedSourcesToApply = append(volumeProjectedSourcesToApply,sourceToApply)
				}
			}
			// append to the projected sources slice that will be update in the deployment
			v.VolumeSource.Projected.Sources = append(v.VolumeSource.Projected.Sources, volumeProjectedSourcesToApply...)
		} else {
			count++
			//deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volumes...)
		}
	}

	if count > 0 {
		r.DeploymentRL.Spec.Template.Spec.Volumes = append(r.DeploymentRL.Spec.Template.Spec.Volumes, volumes...)
		r.DeploymentRL.Spec.Template.Spec.Containers[0].VolumeMounts = append(r.DeploymentRL.Spec.Template.Spec.Containers[0].VolumeMounts, defaultVolumeMount...)
	}

	return nil

}
