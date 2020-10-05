package controllers

import (
	"context"
	"encoding/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
)

type RateLimitDescriptor struct {
	RequestsPerUnit uint32 `yaml:"requests_per_unit"`
	Unit            string `yaml:"unit"`
}

type Descriptors struct {
	Key       string              `yaml:"key"`
	RateLimit RateLimitDescriptor `yaml:"rate_limit"`
	Value     string              `yaml:"value"`
}

type DescriptorsParent struct {
	Descriptors []Descriptors `yaml:"descriptors"`
	Key         string        `yaml:"key"`
}

type ConfigMaptoYAML struct {
	DescriptorsParent []DescriptorsParent `yaml:"descriptors"`
	Domain            string              `yaml:"domain"`
}

func getConfigObjectMatch(typeConfigObjectMatch string, operation string, clusterEndpoint string, context string, nameVhost string) istio_v1alpha3.EnvoyConfigObjectMatch {

	Match := istio_v1alpha3.EnvoyConfigObjectMatch{}

	if typeConfigObjectMatch == "Listener" {

		Match = istio_v1alpha3.EnvoyConfigObjectMatch{
			Context: context,
			Listener: &istio_v1alpha3.ListenerMatch{
				FilterChain: istio_v1alpha3.ListenerMatch_FilterChainMatch{
					Filter: istio_v1alpha3.ListenerMatch_FilterMatch{
						Name: "envoy.http_connection_manager",
						SubFilter: istio_v1alpha3.ListenerMatch_SubFilterMatch{
							Name: "envoy.router",
						},
					},
				},
			},
		}

	}

	if typeConfigObjectMatch == "Cluster" {

		Match = istio_v1alpha3.EnvoyConfigObjectMatch{
			Cluster: &istio_v1alpha3.ClusterMatch{
				Service: clusterEndpoint,
			},
		}

	}

	if typeConfigObjectMatch == "RouteConfiguration" {

		Match = istio_v1alpha3.EnvoyConfigObjectMatch{
			Context: context,
			RouteConfiguration: &istio_v1alpha3.RouteConfigurationMatch{
				Vhost: istio_v1alpha3.RouteConfigurationMatch_VirtualHostMatch{
					Name: nameVhost,
					Route: istio_v1alpha3.RouteConfigurationMatch_RouteMatch{
						Action: "ANY",
					},
				},
			},
		}

	}

	return Match

}

func getEnvoyFilterConfigPatches(applyTo string, operation string, rawConfig json.RawMessage, typeConfigObjectMatch string, clusterEndpoint string, context string, nameVhost string) []istio_v1alpha3.EnvoyConfigObjectPatch {

	ConfigPatches := []istio_v1alpha3.EnvoyConfigObjectPatch{
		{
			ApplyTo: applyTo,
			Patch: istio_v1alpha3.Patch{
				Operation: operation,
				Value:     rawConfig,
			},
			Match: getConfigObjectMatch(typeConfigObjectMatch, operation, clusterEndpoint, context, nameVhost),
		},
	}

	return ConfigPatches

}

func (e EnvoyFilterObject) composeEnvoyFilter(name string, namespace string) istio_v1alpha3.EnvoyFilter {

	envoyFilterBaseDesired := istio_v1alpha3.EnvoyFilter{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EnvoyFilter",
			APIVersion: "networking.istio.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: istio_v1alpha3.EnvoyFilterSpec{
			WorkloadSelector: istio_v1alpha3.WorkloadSelector{
				Labels: e.Labels,
			},
			ConfigPatches: getEnvoyFilterConfigPatches(e.ApplyTo, e.Operation, e.RawConfig, e.TypeConfigObjectMatch, e.ClusterEndpoint, e.Context, e.NameVhost),
		},
	}

	return envoyFilterBaseDesired

}

func (r *RateLimitReconciler) getEnvoyFilter(name string, namespace string) *istio_v1alpha3.EnvoyFilter {

	envoyFilter := istio_v1alpha3.EnvoyFilter{}

	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, &envoyFilter)
	if err != nil {
		klog.Infof("Cannot Found EnvoyFilter %s. Error %v", envoyFilter.Name, err)
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
		klog.Infof("Cannot Found configMap %s. Error %v", found.Name, err)
		return found, err
	}

	return found, nil

}

func constructVolumeSources(name string) []v1.VolumeProjection {

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

	var defaultMode int32

	defaultMode = 0420

	p := &defaultMode

	sources := constructVolumeSources(nameVolumeSource)

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

func (r *RateLimitReconciler) getDeployment(namespace string, name string) (*appsv1.Deployment, error) {

	deploy := &appsv1.Deployment{}
	err := r.Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, deploy)
	if err != nil {
		klog.Infof("Cannot Get Deployment %s. Error %v", "istio-system-ratelimit", err)
		return nil, err
	}
	return deploy, nil
}
