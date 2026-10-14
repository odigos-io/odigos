package k8sconfig

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type pubSubTestDestination struct {
	config.ExporterConfigurer
	secret *corev1.LocalObjectReference
}

func (d pubSubTestDestination) GetSecretRef() *corev1.LocalObjectReference { return d.secret }

func TestPubSubApplicationCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "pubsub", Namespace: "odigos-system"},
		Data:       map[string][]byte{gcpApplicationCredentialsKey: []byte("credentials")},
	}
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	configer := LoadK8sConfigers()[common.GoogleCloudPubSubDestinationType]
	require.NotNil(t, configer)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "odigos-system"},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: k8sconsts.OdigosClusterCollectorContainerName}},
		}}},
	}
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient,
		pubSubTestDestination{}, deployment))
	require.Empty(t, deployment.Spec.Template.Spec.Volumes)
	dest := pubSubTestDestination{secret: &corev1.LocalObjectReference{Name: "pubsub"}}
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, deployment))
	pod := deployment.Spec.Template.Spec
	require.Len(t, pod.Volumes, 1)
	require.Equal(t, "pubsub", pod.Volumes[0].Secret.SecretName)
	require.Equal(t, gcpApplicationCredentialsKey, pod.Volumes[0].Secret.Items[0].Key)
	require.Equal(t, []corev1.EnvVar{{Name: gcpApplicationCredentialsEnvVar, Value: "/secrets/GCP_APPLICATION_CREDENTIALS"}}, pod.Containers[0].Env)
	require.Len(t, pod.Containers[0].VolumeMounts, 1)
	require.ErrorContains(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, deployment), "Only one GCP Destination")
}
