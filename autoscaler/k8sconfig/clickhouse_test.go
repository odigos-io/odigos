package k8sconfig

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

// clickhouseCaSecretName is the secret the destination points at; the configer must read the CA
// from it and mount that same secret.
const clickhouseCaSecretName = "odigos-destination-clickhouse-secret"

func clickhouseCaSecret(t *testing.T, namespace string) *corev1.Secret {
	t.Helper()

	return k8sConfigSecret(clickhouseCaSecretName, namespace, map[string][]byte{
		"CLICKHOUSE_CA_PEM": caCertificatePEM(t),
	})
}

func TestValidateCertificatePem(t *testing.T) {
	validCert := caCertificatePEM(t)

	// a private key block: the right PEM envelope, the wrong contents. Users paste this by mistake.
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	privateKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	// the CERTIFICATE envelope around something that is not a DER certificate
	notACertificatePem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("not der")})

	tests := []struct {
		name    string
		pemData []byte
		wantErr string
	}{
		{
			name:    "a valid certificate is accepted",
			pemData: validCert,
		},
		{
			name:    "a certificate bundle is accepted, the first block is the one validated",
			pemData: append(append([]byte{}, validCert...), validCert...),
		},
		{
			name:    "an empty value is rejected",
			pemData: nil,
			wantErr: "failed to decode PEM block",
		},
		{
			name:    "a value that is not PEM at all is rejected",
			pemData: []byte("-----BEGIN CERTIFICATE-----\nthis is not base64\n"),
			wantErr: "failed to decode PEM block",
		},
		{
			name:    "a PEM block that is not a certificate is rejected",
			pemData: privateKeyPem,
			wantErr: "PEM block is not a certificate",
		},
		{
			name:    "a certificate block that does not parse is rejected",
			pemData: notACertificatePem,
			wantErr: "failed to parse certificate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCertificatePem(tt.pemData)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestClickhouseDestType(t *testing.T) {
	assert.Equal(t, common.DestinationType("clickhouse"), (&Clickhouse{}).DestType())
}

// A destination that does not carry a secret must leave the gateway deployment completely alone;
// the deployment is shared by every destination, so a spurious volume breaks all of them.
func TestClickhouseModifyGatewayCollectorDeployment_NoSecretRef(t *testing.T) {
	noSecretRef := k8sConfigDestination("ch", common.ClickhouseDestinationType, "", nil)
	emptySecretName := k8sConfigDestination("ch", common.ClickhouseDestinationType, "", nil)
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
			dest := tt.dest

			dep := gatewayCollectorDeployment()
			require.NoError(t, (&Clickhouse{}).ModifyGatewayCollectorDeployment(
				context.Background(), newK8sConfigClient(t), dest, dep))

			assert.Equal(t, gatewayCollectorDeployment(), dep)
		})
	}
}

// A secret that the autoscaler cannot read must surface as an error rather than as a deployment
// that silently has no CA mounted, which would fail TLS at runtime with no explanation.
func TestClickhouseModifyGatewayCollectorDeployment_SecretNotFound(t *testing.T) {
	dest := k8sConfigDestination("ch", common.ClickhouseDestinationType, clickhouseCaSecretName, nil)
	dep := gatewayCollectorDeployment()

	err := (&Clickhouse{}).ModifyGatewayCollectorDeployment(
		context.Background(), newK8sConfigClient(t), dest, dep)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret "+clickhouseCaSecretName)
	assert.Equal(t, gatewayCollectorDeployment(), dep)
}

// The secret is looked up in the deployment's namespace. A configer that read it from the
// destination's namespace instead would work by accident in the default install and break for
// anyone running odigos in a different namespace.
func TestClickhouseModifyGatewayCollectorDeployment_ReadsTheSecretFromTheDeploymentNamespace(t *testing.T) {
	dest := k8sConfigDestination("ch", common.ClickhouseDestinationType, clickhouseCaSecretName, nil)
	k8sClient := newK8sConfigClient(t, clickhouseCaSecret(t, "some-other-namespace"))

	dep := gatewayCollectorDeployment()
	err := (&Clickhouse{}).ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret "+clickhouseCaSecretName)
}

// TLS without a custom CA is a supported configuration, so a secret that holds only the password
// must not be treated as an error and must not mount anything.
func TestClickhouseModifyGatewayCollectorDeployment_SecretWithoutCaPem(t *testing.T) {
	tests := []struct {
		name string
		data map[string][]byte
	}{
		{name: "a secret with no data", data: nil},
		{
			name: "a secret that only holds the password",
			data: map[string][]byte{"CLICKHOUSE_PASSWORD": []byte("s3cret")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := k8sConfigDestination("ch", common.ClickhouseDestinationType, clickhouseCaSecretName, nil)
			k8sClient := newK8sConfigClient(t, k8sConfigSecret(clickhouseCaSecretName, k8sConfigNamespace, tt.data))

			dep := gatewayCollectorDeployment()
			require.NoError(t, (&Clickhouse{}).ModifyGatewayCollectorDeployment(
				context.Background(), k8sClient, dest, dep))

			assert.Equal(t, gatewayCollectorDeployment(), dep)
		})
	}
}

// A CA the collector cannot load must fail the destination before the deployment is touched. A
// half-modified deployment (a volume mount with no matching volume) is rejected by the API server,
// which would take the gateway down for every destination.
func TestClickhouseModifyGatewayCollectorDeployment_InvalidCaPem(t *testing.T) {
	dest := k8sConfigDestination("ch-prod", common.ClickhouseDestinationType, clickhouseCaSecretName, nil)
	k8sClient := newK8sConfigClient(t, k8sConfigSecret(clickhouseCaSecretName, k8sConfigNamespace,
		map[string][]byte{"CLICKHOUSE_CA_PEM": []byte("-----BEGIN CERTIFICATE-----\nnope\n")}))

	dep := gatewayCollectorDeployment()
	err := (&Clickhouse{}).ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CA certificate for ClickHouse destination ch-prod")
	assert.Contains(t, err.Error(), "failed to decode PEM block")
	assert.Equal(t, gatewayCollectorDeployment(), dep)
}

func TestClickhouseModifyGatewayCollectorDeployment_GatewayContainerMissing(t *testing.T) {
	dest := k8sConfigDestination("ch", common.ClickhouseDestinationType, clickhouseCaSecretName, nil)
	k8sClient := newK8sConfigClient(t, clickhouseCaSecret(t, k8sConfigNamespace))

	dep := gatewayCollectorDeployment()
	dep.Spec.Template.Spec.Containers = []corev1.Container{{Name: k8sConfigSidecarName}}

	err := (&Clickhouse{}).ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gateway collector container 'gateway' not found")
	assert.Empty(t, dep.Spec.Template.Spec.Volumes)
}

func TestClickhouseModifyGatewayCollectorDeployment_MountsTheCaSecret(t *testing.T) {
	dest := k8sConfigDestination("ch-prod", common.ClickhouseDestinationType, clickhouseCaSecretName, nil)
	k8sClient := newK8sConfigClient(t, clickhouseCaSecret(t, k8sConfigNamespace))

	dep := gatewayCollectorDeployment()
	require.NoError(t, (&Clickhouse{}).ModifyGatewayCollectorDeployment(
		context.Background(), k8sClient, dest, dep))

	assert.Equal(t, []corev1.VolumeMount{{
		Name:      "clickhouse-ca-cert-ch-prod",
		MountPath: "/etc/clickhouse/certs/ch-prod",
		ReadOnly:  true,
	}}, containerNamed(t, dep, k8sConfigGatewayContainerName).VolumeMounts)

	assert.Equal(t, []corev1.Volume{{
		Name: "clickhouse-ca-cert-ch-prod",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: clickhouseCaSecretName,
				Items: []corev1.KeyToPath{{
					Key:  "CLICKHOUSE_CA_PEM",
					Path: "CLICKHOUSE_CA_PEM",
				}},
			},
		},
	}}, dep.Spec.Template.Spec.Volumes)

	// the CA belongs to the gateway container only
	assert.Empty(t, containerNamed(t, dep, k8sConfigSidecarName).VolumeMounts)
}

// Destination IDs are derived from the destination name and routinely contain dots. Kubernetes
// volume names are DNS labels and reject dots, so the name is sanitized while the mount path -
// which the exporter config refers to verbatim - must keep the ID unchanged.
func TestClickhouseModifyGatewayCollectorDeployment_SanitizesDotsInTheVolumeName(t *testing.T) {
	dest := k8sConfigDestination("clickhouse.prod.example.com", common.ClickhouseDestinationType, clickhouseCaSecretName, nil)
	k8sClient := newK8sConfigClient(t, clickhouseCaSecret(t, k8sConfigNamespace))

	dep := gatewayCollectorDeployment()
	require.NoError(t, (&Clickhouse{}).ModifyGatewayCollectorDeployment(
		context.Background(), k8sClient, dest, dep))

	mounts := containerNamed(t, dep, k8sConfigGatewayContainerName).VolumeMounts
	require.Len(t, mounts, 1)
	assert.Equal(t, "clickhouse-ca-cert-clickhouse-prod-example-com", mounts[0].Name)
	assert.Equal(t, "/etc/clickhouse/certs/clickhouse.prod.example.com", mounts[0].MountPath)

	require.Len(t, dep.Spec.Template.Spec.Volumes, 1)
	assert.Equal(t, mounts[0].Name, dep.Spec.Template.Spec.Volumes[0].Name)
}

// Reconciles are repeated, so the same destination must never accumulate duplicate mounts - the
// API server rejects a duplicate volume name and the gateway stops being updated.
func TestClickhouseModifyGatewayCollectorDeployment_IsIdempotent(t *testing.T) {
	dest := k8sConfigDestination("ch-prod", common.ClickhouseDestinationType, clickhouseCaSecretName, nil)
	k8sClient := newK8sConfigClient(t, clickhouseCaSecret(t, k8sConfigNamespace))

	dep := gatewayCollectorDeployment()
	configer := &Clickhouse{}
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep))
	afterFirstCall := dep.DeepCopy()

	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient, dest, dep))

	assert.Equal(t, afterFirstCall, dep)
}

// The volume and the volume mount are deduplicated independently, so a deployment that already
// carries the volume still needs the mount added - and must not end up with the volume twice.
func TestClickhouseModifyGatewayCollectorDeployment_AddsTheMountWhenOnlyTheVolumeExists(t *testing.T) {
	dest := k8sConfigDestination("ch-prod", common.ClickhouseDestinationType, clickhouseCaSecretName, nil)
	k8sClient := newK8sConfigClient(t, clickhouseCaSecret(t, k8sConfigNamespace))

	dep := gatewayCollectorDeployment()
	dep.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: "clickhouse-ca-cert-ch-prod"}}

	require.NoError(t, (&Clickhouse{}).ModifyGatewayCollectorDeployment(
		context.Background(), k8sClient, dest, dep))

	assert.Len(t, containerNamed(t, dep, k8sConfigGatewayContainerName).VolumeMounts, 1)
	// the pre-existing volume is kept as it is rather than appended a second time
	assert.Equal(t, []corev1.Volume{{Name: "clickhouse-ca-cert-ch-prod"}}, dep.Spec.Template.Spec.Volumes)
}

// Several ClickHouse destinations share one gateway. Each needs its own volume, its own mount path
// and its own secret, or one destination's CA silently overwrites another's.
func TestClickhouseModifyGatewayCollectorDeployment_SupportsMultipleDestinations(t *testing.T) {
	secondSecretName := "odigos-destination-clickhouse-staging-secret"
	k8sClient := newK8sConfigClient(t,
		clickhouseCaSecret(t, k8sConfigNamespace),
		k8sConfigSecret(secondSecretName, k8sConfigNamespace, map[string][]byte{
			"CLICKHOUSE_CA_PEM": caCertificatePEM(t),
		}),
	)

	dep := gatewayCollectorDeployment()
	configer := &Clickhouse{}
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient,
		k8sConfigDestination("ch-prod", common.ClickhouseDestinationType, clickhouseCaSecretName, nil), dep))
	require.NoError(t, configer.ModifyGatewayCollectorDeployment(context.Background(), k8sClient,
		k8sConfigDestination("ch-staging", common.ClickhouseDestinationType, secondSecretName, nil), dep))

	mounts := containerNamed(t, dep, k8sConfigGatewayContainerName).VolumeMounts
	require.Len(t, mounts, 2)
	assert.Equal(t, "clickhouse-ca-cert-ch-prod", mounts[0].Name)
	assert.Equal(t, "/etc/clickhouse/certs/ch-prod", mounts[0].MountPath)
	assert.Equal(t, "clickhouse-ca-cert-ch-staging", mounts[1].Name)
	assert.Equal(t, "/etc/clickhouse/certs/ch-staging", mounts[1].MountPath)

	volumes := dep.Spec.Template.Spec.Volumes
	require.Len(t, volumes, 2)
	assert.Equal(t, clickhouseCaSecretName, volumes[0].Secret.SecretName)
	assert.Equal(t, secondSecretName, volumes[1].Secret.SecretName)
}

// The exporter config and the volume mount are produced by two different packages that never call
// each other, and nothing at runtime reports a mismatch: the collector just fails to load the CA.
// Rebuild the path the CA actually lands on from the mount and compare it to the ca_file the
// exporter is told to read.
func TestClickhouseCaFileMatchesTheMountedPath(t *testing.T) {
	const destID = "ch-prod"
	dest := k8sConfigDestination(destID, common.ClickhouseDestinationType, clickhouseCaSecretName, map[string]string{
		"CLICKHOUSE_ENDPOINT":      "clickhouse.example.com:9440",
		"CLICKHOUSE_DATABASE_NAME": "otel",
		"CLICKHOUSE_TLS_ENABLED":   "true",
		"CLICKHOUSE_USE_CUSTOM_CA": "true",
	})

	gatewayConfig := &config.Config{
		Exporters:  config.GenericMap{},
		Processors: config.GenericMap{},
		Service:    config.Service{Pipelines: map[string]config.Pipeline{}},
	}
	_, err := (&config.Clickhouse{}).ModifyConfig(dest, gatewayConfig)
	require.NoError(t, err)

	exporter, ok := gatewayConfig.Exporters["clickhouse/clickhouse-"+destID].(config.GenericMap)
	require.True(t, ok, "the clickhouse exporter was not rendered")
	tlsConfig, ok := exporter["tls"].(config.GenericMap)
	require.True(t, ok, "the clickhouse exporter has no tls config")
	caFile, ok := tlsConfig["ca_file"].(string)
	require.True(t, ok, "the clickhouse exporter has no ca_file")

	dep := gatewayCollectorDeployment()
	require.NoError(t, (&Clickhouse{}).ModifyGatewayCollectorDeployment(
		context.Background(), newK8sConfigClient(t, clickhouseCaSecret(t, k8sConfigNamespace)), dest, dep))

	mounts := containerNamed(t, dep, k8sConfigGatewayContainerName).VolumeMounts
	require.Len(t, mounts, 1)
	volumes := dep.Spec.Template.Spec.Volumes
	require.Len(t, volumes, 1)
	require.NotNil(t, volumes[0].Secret)
	require.Len(t, volumes[0].Secret.Items, 1)
	assert.Equal(t, mounts[0].Name, volumes[0].Name, "the mount refers to a volume that does not exist")

	mountedCaPath := mounts[0].MountPath + "/" + volumes[0].Secret.Items[0].Path
	assert.Equal(t, caFile, mountedCaPath,
		"the exporter reads the CA from a path the gateway does not mount it on")
}
