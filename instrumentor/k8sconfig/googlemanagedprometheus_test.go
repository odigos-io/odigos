package k8sconfig

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type gmpK8sDestination struct{ secret *corev1.LocalObjectReference }

func (g gmpK8sDestination) GetID() string { return "test" }
func (g gmpK8sDestination) GetType() common.DestinationType {
	return common.GoogleManagedPrometheusDestinationType
}
func (g gmpK8sDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.MetricsObservabilitySignal}
}
func (g gmpK8sDestination) GetConfig() map[string]string               { return nil }
func (g gmpK8sDestination) GetSecretRef() *corev1.LocalObjectReference { return g.secret }

func TestGoogleManagedPrometheusCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "gmp", Namespace: "odigos-system"}, Data: map[string][]byte{gcpApplicationCredentialsKey: []byte("test-credentials")}}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	configer := LoadK8sConfigers()[common.GoogleManagedPrometheusDestinationType]
	require.NotNil(t, configer)
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "odigos-system"}}
	dep.Spec.Template.Spec.Containers = []corev1.Container{{Name: k8sconsts.OdigosClusterCollectorContainerName}}
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), client, gmpK8sDestination{}, dep))
	require.Empty(t, dep.Spec.Template.Spec.Volumes)
	dest := gmpK8sDestination{secret: &corev1.LocalObjectReference{Name: "gmp"}}
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), client, dest, dep))
	require.Len(t, dep.Spec.Template.Spec.Volumes, 1)
	require.Equal(t, "gmp", dep.Spec.Template.Spec.Volumes[0].Secret.SecretName)
	require.Equal(t, gcpApplicationCredentialsKey, dep.Spec.Template.Spec.Volumes[0].Secret.Items[0].Key)
	require.Contains(t, dep.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: gcpApplicationCredentialsEnvVar, Value: "/secrets/GCP_APPLICATION_CREDENTIALS"})
	require.ErrorContains(t, configer.ModifyGatewayCollectorDeployment(context.Background(), client, dest, dep), "Only one GCP Destination")
}
