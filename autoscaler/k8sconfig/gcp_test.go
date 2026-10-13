package k8sconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

const gcpCredentialsSecretName = "odigos-destination-googlecloud-secret"

func gcpCredentialsSecret(namespace string) *corev1.Secret {
	return k8sConfigSecret(gcpCredentialsSecretName, namespace, map[string][]byte{
		"GCP_APPLICATION_CREDENTIALS": []byte(`{"type":"service_account"}`),
	})
}

func TestGoogleCloudDestType(t *testing.T) {
	assert.Equal(t, common.DestinationType("googlecloud"), (&GoogleCloud{}).DestType())
}

// Running on GCP with workload identity needs no credentials file, which is the common case, so a
// destination with no secret must leave the shared gateway deployment untouched.
func TestGCPModifyGatewayCollectorDeployment_NoSecretRef(t *testing.T) {
	noSecretRef := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, "", nil)
	emptySecretName := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, "", nil)
	emptySecretName.Spec.SecretRef = &corev1.LocalObjectReference{Name: ""}

	tests := []struct {
		name string
		dest odigosv1.Destination
	}{
		{name: "no secret ref at all", dest: noSecretRef},
		{name: "a secret ref with an empty name", dest: emptySecretName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := gatewayCollectorDeployment()
			require.NoError(t, (&GoogleCloud{}).ModifyGatewayCollectorDeployment(
				context.Background(), newK8sConfigClient(t), tt.dest, dep))

			assert.Equal(t, gatewayCollectorDeployment(), dep)
		})
	}
}

func TestGCPModifyGatewayCollectorDeployment_SecretNotFound(t *testing.T) {
	dest := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, gcpCredentialsSecretName, nil)
	dep := gatewayCollectorDeployment()

	err := (&GoogleCloud{}).ModifyGatewayCollectorDeployment(
		context.Background(), newK8sConfigClient(t), dest, dep)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret "+gcpCredentialsSecretName)
	assert.Equal(t, gatewayCollectorDeployment(), dep)
}

// The secret is read from the deployment's namespace, not from a namespace of the configer's own.
func TestGCPModifyGatewayCollectorDeployment_ReadsTheSecretFromTheDeploymentNamespace(t *testing.T) {
	dest := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, gcpCredentialsSecretName, nil)
	k8sClient := newK8sConfigClient(t, gcpCredentialsSecret("some-other-namespace"))

	dep := gatewayCollectorDeployment()
	err := (&GoogleCloud{}).ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret "+gcpCredentialsSecretName)
}

// A GCP destination secret can legitimately hold other keys; only the application credentials key
// triggers the mount, and its absence is not an error.
func TestGCPModifyGatewayCollectorDeployment_SecretWithoutApplicationCredentials(t *testing.T) {
	tests := []struct {
		name string
		data map[string][]byte
	}{
		{name: "a secret with no data", data: nil},
		{
			name: "a secret that holds an unrelated key",
			data: map[string][]byte{"GCP_PROJECT_ID": []byte("my-project")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, gcpCredentialsSecretName, nil)
			k8sClient := newK8sConfigClient(t, k8sConfigSecret(gcpCredentialsSecretName, k8sConfigNamespace, tt.data))

			dep := gatewayCollectorDeployment()
			require.NoError(t, (&GoogleCloud{}).ModifyGatewayCollectorDeployment(
				context.Background(), k8sClient, dest, dep))

			assert.Equal(t, gatewayCollectorDeployment(), dep)
		})
	}
}

func TestGCPModifyGatewayCollectorDeployment_GatewayContainerMissing(t *testing.T) {
	dest := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, gcpCredentialsSecretName, nil)
	k8sClient := newK8sConfigClient(t, gcpCredentialsSecret(k8sConfigNamespace))

	dep := gatewayCollectorDeployment()
	dep.Spec.Template.Spec.Containers = []corev1.Container{{Name: k8sConfigSidecarName}}

	err := (&GoogleCloud{}).ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gateway collector container 'gateway' not found")
	assert.Empty(t, dep.Spec.Template.Spec.Volumes)
}

// The collector exporter finds the credentials only through GOOGLE_APPLICATION_CREDENTIALS, so the
// env var value has to be exactly the path the secret key is projected onto.
func TestGCPModifyGatewayCollectorDeployment_MountsCredentialsAndSetsTheEnvVar(t *testing.T) {
	dest := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, gcpCredentialsSecretName, nil)
	k8sClient := newK8sConfigClient(t, gcpCredentialsSecret(k8sConfigNamespace))

	dep := gatewayCollectorDeployment()
	require.NoError(t, (&GoogleCloud{}).ModifyGatewayCollectorDeployment(
		context.Background(), k8sClient, dest, dep))

	gateway := containerNamed(t, dep, k8sConfigGatewayContainerName)
	assert.Equal(t, []corev1.VolumeMount{{
		Name:      "gcp-credentials-secret",
		MountPath: "/secrets",
	}}, gateway.VolumeMounts)

	assert.Equal(t, []corev1.EnvVar{{
		Name:  "GOOGLE_APPLICATION_CREDENTIALS",
		Value: "/secrets/GCP_APPLICATION_CREDENTIALS",
	}}, gateway.Env)

	assert.Equal(t, []corev1.Volume{{
		Name: "gcp-credentials-secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: gcpCredentialsSecretName,
				Items: []corev1.KeyToPath{{
					Key:  "GCP_APPLICATION_CREDENTIALS",
					Path: "GCP_APPLICATION_CREDENTIALS",
				}},
			},
		},
	}}, dep.Spec.Template.Spec.Volumes)

	sidecar := containerNamed(t, dep, k8sConfigSidecarName)
	assert.Empty(t, sidecar.VolumeMounts)
	assert.Empty(t, sidecar.Env)
}

// Existing env vars and volumes on the gateway must survive; the credentials are appended.
func TestGCPModifyGatewayCollectorDeployment_KeepsExistingVolumesAndEnv(t *testing.T) {
	dest := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, gcpCredentialsSecretName, nil)
	k8sClient := newK8sConfigClient(t, gcpCredentialsSecret(k8sConfigNamespace))

	dep := gatewayCollectorDeployment()
	dep.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: "collector-conf"}}
	dep.Spec.Template.Spec.Containers[1].VolumeMounts = []corev1.VolumeMount{{
		Name: "collector-conf", MountPath: "/conf",
	}}
	dep.Spec.Template.Spec.Containers[1].Env = []corev1.EnvVar{{Name: "GOMEMLIMIT", Value: "340MiB"}}

	require.NoError(t, (&GoogleCloud{}).ModifyGatewayCollectorDeployment(
		context.Background(), k8sClient, dest, dep))

	gateway := containerNamed(t, dep, k8sConfigGatewayContainerName)
	require.Len(t, gateway.VolumeMounts, 2)
	assert.Equal(t, "collector-conf", gateway.VolumeMounts[0].Name)
	assert.Equal(t, "gcp-credentials-secret", gateway.VolumeMounts[1].Name)

	require.Len(t, gateway.Env, 2)
	assert.Equal(t, "GOMEMLIMIT", gateway.Env[0].Name)
	assert.Equal(t, "GOOGLE_APPLICATION_CREDENTIALS", gateway.Env[1].Name)

	require.Len(t, dep.Spec.Template.Spec.Volumes, 2)
	assert.Equal(t, "collector-conf", dep.Spec.Template.Spec.Volumes[0].Name)
	assert.Equal(t, "gcp-credentials-secret", dep.Spec.Template.Spec.Volumes[1].Name)
}

// GOOGLE_APPLICATION_CREDENTIALS is a single process-wide env var, so a second GCP destination with
// its own credentials cannot be honoured. Unlike the ClickHouse configer this one is deliberately
// not idempotent: it reports a conflict instead of quietly using the first destination's
// credentials for both. Each of the three guards has to fire on its own, because a deployment can
// reach any one of these states.
func TestGCPModifyGatewayCollectorDeployment_RejectsASecondCredentialsFile(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(dep *appsv1.Deployment)
		wantErr string
	}{
		{
			name: "the volume mount is already there",
			prepare: func(dep *appsv1.Deployment) {
				dep.Spec.Template.Spec.Containers[1].VolumeMounts = []corev1.VolumeMount{{
					Name: "gcp-credentials-secret", MountPath: "/secrets",
				}}
			},
			wantErr: "GCP credentials volume mount gcp-credentials-secret already exists",
		},
		{
			name: "only the volume is already there",
			prepare: func(dep *appsv1.Deployment) {
				dep.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: "gcp-credentials-secret"}}
			},
			wantErr: "GCP credentials volume gcp-credentials-secret already exists",
		},
		{
			name: "only the environment variable is already there",
			prepare: func(dep *appsv1.Deployment) {
				dep.Spec.Template.Spec.Containers[1].Env = []corev1.EnvVar{{
					Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/secrets/GCP_APPLICATION_CREDENTIALS",
				}}
			},
			wantErr: "GCP credentials environment variable GOOGLE_APPLICATION_CREDENTIALS already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, gcpCredentialsSecretName, nil)
			k8sClient := newK8sConfigClient(t, gcpCredentialsSecret(k8sConfigNamespace))

			dep := gatewayCollectorDeployment()
			tt.prepare(dep)

			err := (&GoogleCloud{}).ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Contains(t, err.Error(),
				"Only one GCP Destination may have Application Credentials configured")

			// whichever guard fired, nothing may be duplicated
			assert.LessOrEqual(t, len(dep.Spec.Template.Spec.Volumes), 1)
			gateway := containerNamed(t, dep, k8sConfigGatewayContainerName)
			assert.LessOrEqual(t, len(gateway.VolumeMounts), 1)
			assert.LessOrEqual(t, len(gateway.Env), 1)
		})
	}
}

// The same destination reconciled twice against the same deployment object hits the conflict
// guards. In production this does not happen because clustercollector rebuilds the deployment from
// scratch on every reconcile, so pin that assumption: if the deployment were ever reused, GCP
// destinations would break.
func TestGCPModifyGatewayCollectorDeployment_IsNotIdempotent(t *testing.T) {
	dest := k8sConfigDestination("gcp", common.GoogleCloudDestinationType, gcpCredentialsSecretName, nil)
	k8sClient := newK8sConfigClient(t, gcpCredentialsSecret(k8sConfigNamespace))

	dep := gatewayCollectorDeployment()
	configer := &GoogleCloud{}
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep))

	err := configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "GCP credentials volume mount gcp-credentials-secret already exists")
}
