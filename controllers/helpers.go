package controllers

import (
	"encoding/json"
	"fmt"

	"context"
	"k8s.io/apimachinery/pkg/types"

	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (r *RateLimitReconciler) desiredConfigMap(rateLimitInstance *networkingv1alpha1.RateLimit, desiredNamespace string, name string) (v1.ConfigMap, error) {

	configMapData := make(map[string]string)

	configyaml := ConfigMaptoYAML{}

	for _, dimension := range rateLimitInstance.Spec.Dimensions {
		// we assume the second dimension is always destination_cluster
		for k, ratelimitdimension := range dimension {
			fmt.Printf("%s -> %s\n", k, ratelimitdimension)
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
		//return ctrl.Result{}, err
		fmt.Println("not found")
		return &envoyFilter
	}

	return &envoyFilter

}


func (r *RateLimitReconciler) getConfigMap(name string, namespace string) (v1.ConfigMap, error ) {

	found := v1.ConfigMap{}


	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, &found)
	if err != nil {
		//return ctrl.Result{}, err
		fmt.Println("not found")
		return found, err
	}

	return found, nil




}
