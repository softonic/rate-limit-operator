package controllers

import (
	"fmt"
	"github.com/gogo/protobuf/jsonpb"
	protobuftypes "github.com/gogo/protobuf/types"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
	istio "istio.io/api/networking/v1alpha3"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "strconv"
	"strings"
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

func (r *RateLimitReconciler) desiredConfigMap(rateLimitInstance *networkingv1alpha1.RateLimit, desiredNamespace string) (v1.ConfigMap, error) {

	configMapData := make(map[string]string)

	// test.yaml: |-
	// descriptors:
	// - descriptors:
	//   - key: destination_cluster
	//     rate_limit:
	//       requests_per_unit: 5
	//       unit: minute
	//     value: outbound|80||chicken-head-nginx.chicken-head.svc.cluster.local
	//   key: remote_address
	// domain: test

	configyaml := ConfigMaptoYAML{}

	for _, dimension := range rateLimitInstance.Spec.Dimensions {
		// we assume the second dimension is always destination_cluster ??
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
						Domain: "test",
					}
				}
			}
		}
	}

	configYamlFile, _ := yaml.Marshal(&configyaml)

	configMapData["test.yaml"] = string(configYamlFile)

	configMap := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: desiredNamespace,
		},
		Data: configMapData,
	}

	return configMap, nil

}

func buildClusterPatches(config string) []*istio.EnvoyFilter_EnvoyConfigObjectPatch {
	val := &protobuftypes.Struct{}
	jsonpb.Unmarshal(strings.NewReader(config), val)

	return []*istio.EnvoyFilter_EnvoyConfigObjectPatch{
		{
			ApplyTo: istio.EnvoyFilter_CLUSTER,
			Patch: &istio.EnvoyFilter_Patch{
				Operation: istio.EnvoyFilter_Patch_ADD,
				Value:     val,
			},
		},
	}
}
