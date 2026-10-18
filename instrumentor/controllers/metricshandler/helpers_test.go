package metricshandler

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/assert"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

func TestIsOwnedByOdigos(t *testing.T) {
	t.Parallel()

	assert.False(t, IsOwnedByOdigos(&apiregv1.APIService{}))
	assert.False(t, IsOwnedByOdigos(&apiregv1.APIService{
		Spec: apiregv1.APIServiceSpec{Service: &apiregv1.ServiceReference{Name: "keda-operator"}},
	}))
	assert.True(t, IsOwnedByOdigos(&apiregv1.APIService{
		Spec: apiregv1.APIServiceSpec{Service: &apiregv1.ServiceReference{Name: k8sconsts.AutoScalerWebhookServiceName}},
	}))
	assert.True(t, IsOwnedByOdigos(&apiregv1.APIService{
		Spec: apiregv1.APIServiceSpec{Service: &apiregv1.ServiceReference{Name: k8sconsts.InstrumentorServiceName}},
	}))
}
