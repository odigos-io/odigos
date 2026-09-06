package metricshandler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

func TestIsOwnedByOdigos(t *testing.T) {
	for _, tc := range []struct {
		name     string
		service  *apiregv1.ServiceReference
		expected bool
	}{
		{
			// the odigos autoscaler service, spelled out because it is named by the helm chart
			name:     "served by the odigos autoscaler",
			service:  &apiregv1.ServiceReference{Name: "odigos-autoscaler", Namespace: gatewayTestNamespace},
			expected: true,
		},
		{
			name:     "served by another metrics adapter",
			service:  &apiregv1.ServiceReference{Name: "prometheus-adapter", Namespace: "monitoring"},
			expected: false,
		},
		{
			// an APIService may point at an external endpoint instead of a service
			name:     "no service reference at all",
			service:  nil,
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apiSvc := &apiregv1.APIService{
				ObjectMeta: metav1.ObjectMeta{Name: "v1beta1.custom.metrics.k8s.io"},
				Spec:       apiregv1.APIServiceSpec{Service: tc.service},
			}

			assert.Equal(t, tc.expected, IsOwnedByOdigos(apiSvc))
		})
	}
}
