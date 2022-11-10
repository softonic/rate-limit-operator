package controllers

import (
	"context"

	"github.com/softonic/rate-limit-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	networking "istio.io/api/networking/v1alpha3"
	clientIstio "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func (r *RateLimitReconciler) getK8sResources(baseName string, istioNamespace string, controllerNamespace string, deploymentName string) error {

	r.getEnvoyFilters(baseName, istioNamespace)

	var err error

	r.configMapRateLimit, err = r.getConfigMap(baseName, controllerNamespace)
	if err != nil {
		klog.Infof("Cannot Found ConfigMap in the getk8sresource func %s. Error %v", baseName, err)

	}

	return nil
}

func getConfigObjectMatch(typeConfigObjectMatch string, operation networking.EnvoyFilter_Patch_Operation, clusterEndpoint string, context string, nameVhost string, route string) *networking.EnvoyFilter_EnvoyConfigObjectMatch {

	Match := networking.EnvoyFilter_EnvoyConfigObjectMatch{}

	vhost := networking.EnvoyFilter_RouteConfigurationMatch_VirtualHostMatch{}

	if typeConfigObjectMatch == "Listener" {

		Match = networking.EnvoyFilter_EnvoyConfigObjectMatch{
			Context: networking.EnvoyFilter_SIDECAR_INBOUND,
			ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
				Listener: &networking.EnvoyFilter_ListenerMatch{
					FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
						Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
							Name: "envoy.filters.network.http_connection_manager",
							SubFilter: &networking.EnvoyFilter_ListenerMatch_SubFilterMatch{
								Name: "envoy.filters.http.router",
							},
						},
					},
				},
			},
		}
	}

	if typeConfigObjectMatch == "Cluster" {

		Match = networking.EnvoyFilter_EnvoyConfigObjectMatch{
			Context: networking.EnvoyFilter_GATEWAY,
			ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Cluster{
				Cluster: &networking.EnvoyFilter_ClusterMatch{
					Service: clusterEndpoint,
				},
			},
		}

	}

	if route != "" {
		vhost = networking.EnvoyFilter_RouteConfigurationMatch_VirtualHostMatch{
			Route: &networking.EnvoyFilter_RouteConfigurationMatch_RouteMatch{
				Action: networking.EnvoyFilter_RouteConfigurationMatch_RouteMatch_ANY,
				Name:   route,
			},
		}
	} else {
		vhost = networking.EnvoyFilter_RouteConfigurationMatch_VirtualHostMatch{
			Name: nameVhost,
			Route: &networking.EnvoyFilter_RouteConfigurationMatch_RouteMatch{
				Action: networking.EnvoyFilter_RouteConfigurationMatch_RouteMatch_ANY,
			},
		}
	}

	if typeConfigObjectMatch == "RouteConfiguration" {

		Match = networking.EnvoyFilter_EnvoyConfigObjectMatch{
			Context: networking.EnvoyFilter_GATEWAY,
			ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration{
				RouteConfiguration: &networking.EnvoyFilter_RouteConfigurationMatch{
					Vhost: &vhost,
				},
			},
		}

	}

	return &Match

}

func getEnvoyFilterConfigPatches(applyTo networking.EnvoyFilter_ApplyTo, operation networking.EnvoyFilter_Patch_Operation, rawConfig string, typeConfigObjectMatch string, clusterEndpoint string, context string, nameVhost string, routes []string) []*networking.EnvoyFilter_EnvoyConfigObjectPatch {

	ConfigPatches := []*networking.EnvoyFilter_EnvoyConfigObjectPatch{}
	element := networking.EnvoyFilter_EnvoyConfigObjectPatch{}

	// value, err := g.buildHttpFilterPatchValue()
	// if err != nil {
	// 	return nil, err
	// }

	// listener, err := g.buildHttpFilterListener()
	// if err != nil {
	// 	return nil, err
	// }

	if len(routes) > 0 {
		for _, route := range routes {
			element = networking.EnvoyFilter_EnvoyConfigObjectPatch{
				ApplyTo: applyTo,
				Patch: &networking.EnvoyFilter_Patch{
					Operation: operation,
					Value:     utils.ConvertYaml2Struct(rawConfig),
				},
				Match: getConfigObjectMatch(typeConfigObjectMatch, operation, clusterEndpoint, context, nameVhost, route),
			}
			ConfigPatches = append(ConfigPatches, &element)
		}
	} else {
		ConfigPatches = []*networking.EnvoyFilter_EnvoyConfigObjectPatch{
			{
				ApplyTo: applyTo,
				Patch: &networking.EnvoyFilter_Patch{
					Operation: operation,
					Value:     utils.ConvertYaml2Struct(rawConfig),
				},
				Match: getConfigObjectMatch(typeConfigObjectMatch, operation, clusterEndpoint, context, nameVhost, ""),
			},
		}
	}

	return ConfigPatches

}

func (e EnvoyFilterObject) composeEnvoyFilter(name string, namespace string) *clientIstio.EnvoyFilter {

	envoyFilterBaseDesired := &clientIstio.EnvoyFilter{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EnvoyFilter",
			APIVersion: "networking.istio.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networking.EnvoyFilter{
			WorkloadSelector: &networking.WorkloadSelector{
				Labels: e.Labels,
			},
			ConfigPatches: getEnvoyFilterConfigPatches(e.ApplyTo, e.Operation, e.RawConfig, e.TypeConfigObjectMatch, e.ClusterEndpoint, e.Context, e.NameVhost, e.Routes),
		},
	}

	return envoyFilterBaseDesired

}

func (r *RateLimitReconciler) getEnvoyFilters(baseName string, istioNamespace string) *[]*clientIstio.EnvoyFilter {

	// case switch with the type of the filter

	envoyFilterCluster := r.getEnvoyFilter(baseName+"-cluster", istioNamespace)

	envoyFilterHTTPFilter := r.getEnvoyFilter(baseName+"-envoy-filter", istioNamespace)

	envoyFilterHTTPRoute := r.getEnvoyFilter(baseName+"-route", istioNamespace)

	r.EnvoyFilters = append(r.EnvoyFilters, envoyFilterCluster, envoyFilterHTTPFilter, envoyFilterHTTPRoute)

	return &r.EnvoyFilters

}

func (r *RateLimitReconciler) getEnvoyFilter(name string, namespace string) *clientIstio.EnvoyFilter {

	envoyFilter := clientIstio.EnvoyFilter{}

	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, &envoyFilter)
	if err != nil {
		klog.Infof("Cannot Found EnvoyFilter %s. Error %v", name, err)
		return &envoyFilter
	}

	return &envoyFilter

}

func (r *RateLimitReconciler) getConfigMap(name string, namespace string) (v1.ConfigMap, error) {

	found := v1.ConfigMap{}

	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, &found)
	if err != nil {
		//klog.Infof("Cannot Found configMap %s. Error %v", found.Name, err)
		return found, err
	}

	return found, nil

}

func constructVolumeSources(name string) []v1.VolumeProjection {

	//sources := make([]v1.VolumeProjection, 0)

	sources := []v1.VolumeProjection{
		{
			ConfigMap: &v1.ConfigMapProjection{
				LocalObjectReference: v1.LocalObjectReference{
					Name: name,
				},
			},
		},
	}

	return sources
}

func constructVolumes(nameVolume string, nameVolumeSource string) []v1.Volume {

	var defaultMode int32 = 0420

	p := &defaultMode

	sources := constructVolumeSources(nameVolumeSource)

	//	Volumes := make([]v1.Volume, 0)

	Volumes := []v1.Volume{
		{
			Name: nameVolume,
			VolumeSource: v1.VolumeSource{
				Projected: &v1.ProjectedVolumeSource{
					DefaultMode: p,
					Sources:     sources,
				},
			},
		},
	}

	return Volumes
}

func (r *RateLimitReconciler) getDeployment(controllerNamespace string, name string) (appsv1.Deployment, error) {

	found := appsv1.Deployment{}

	//klog.Infof("Before getting this deployment")

	deploy := &appsv1.Deployment{}
	err := r.Get(context.TODO(), client.ObjectKey{
		Namespace: controllerNamespace,
		Name:      name,
	}, deploy)
	if err != nil {
		klog.Infof("Cannot Get Deployment %s. Error %v", "istio-system-ratelimit", err)
		return found, err
	}

	//klog.Infof("Getting this deployment %v", deploy)

	return *deploy, nil
}

func (r *RateLimitReconciler) getVirtualService(namespace string, name string) (*clientIstio.VirtualService, error) {

	virtualService := &clientIstio.VirtualService{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, virtualService)
	if err != nil {
		return nil, err
	}

	return virtualService, nil

}

func (r *RateLimitReconciler) getGateway(namespace string, name string) (*clientIstio.Gateway, error) {

	Gateway := &clientIstio.Gateway{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, Gateway)
	if err != nil {
		return nil, err
	}

	return Gateway, nil

}
