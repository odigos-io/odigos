package k8sconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

type gcsTestDestination struct{ secret *corev1.LocalObjectReference }

func (d gcsTestDestination) GetID() string                   { return "gcs-test" }
func (d gcsTestDestination) GetType() common.DestinationType { return common.GCSDestinationType }
func (d gcsTestDestination) GetConfig() map[string]string    { return map[string]string{} }
func (d gcsTestDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.TracesObservabilitySignal}
}
func (d gcsTestDestination) GetSecretRef() *corev1.LocalObjectReference { return d.secret }

func TestGCSApplicationCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "gcs-credentials", Namespace: "odigos-system"}, Data: map[string][]byte{"GCP_APPLICATION_CREDENTIALS": []byte("credentials-json")}}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	configer, exists := LoadK8sConfigers()[common.GCSDestinationType]
	require.True(t, exists)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "odigos-system"}, Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: k8sconsts.OdigosClusterCollectorContainerName}}}}}}
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient, gcsTestDestination{}, deployment))
	require.Empty(t, deployment.Spec.Template.Spec.Volumes)
	require.Empty(t, deployment.Spec.Template.Spec.Containers[0].Env)
	dest := gcsTestDestination{secret: &corev1.LocalObjectReference{Name: secret.Name}}
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, deployment))
	require.Equal(t, secret.Name, deployment.Spec.Template.Spec.Volumes[0].Secret.SecretName)
	require.Equal(t, "/secrets", deployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)
	require.Contains(t, deployment.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/secrets/GCP_APPLICATION_CREDENTIALS"})
	require.ErrorContains(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, deployment), "Only one GCP Destination")
}
