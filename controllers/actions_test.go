package controllers


import (
	"github.com/softonic/rate-limit-operator/api/istio_v1alpha3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	networkingv1alpha1 "github.com/softonic/rate-limit-operator/api/v1alpha1"
	"testing"

)


func TestFailIfGenerateConfigMap(t *testing.T) {


	instance := networkingv1alpha1.RateLimit{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RateLimit",
			APIVersion: "networking.softonic.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance",
			Namespace: "ratelimitoperatortest",
		},
		Spec: networkingv1alpha1.RateLimitSpec{
			TargetRef: v1.ObjectReference{
				Kind: "VirtualService",
				Name: "vs-test",
				APIVersion: "networking.istio.io/v1alpha3",
				Namespace: "ratelimitoperatortest",
			},
			Dimensions: networkingv1alpha1.DimensionsList{
				{
					Key: "remote_address_persecond",
					Descriptors: []networkingv1alpha1.Descriptors{
						{
							Key: "destination_cluster",
							Ratelimits: networkingv1alpha1.RateLimitS{
								RequestsPerUnit: 50,
								Unit: "second",
							},
							Value: "outbound|80||server.noodle-v1.svc.cluster.local",
						},
					},
				},
			},
		},
	}


	// I would like to use method generateConfigMap and see if the configmap generated is correct.
	



	//t.Errorf("Status should be Fail, but got %v", admissionReview.Response.Result.Status)

}