package k8sconfig

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

const (
	// the namespace the gateway collector deployment under test lives in. Deliberately not the
	// default "odigos-system": every secret lookup has to follow the deployment's namespace, and a
	// lookup against a hardcoded default would pass unnoticed in a fixture that used it.
	k8sConfigNamespace = "odigos-custom-ns"
	// a container placed in front of the gateway container in the fixture, so that a test can tell
	// a lookup by container name apart from a lookup that just takes the first container
	k8sConfigSidecarName = "config-reloader"
	// the container the configers are expected to modify, spelled out rather than taken from
	// k8sconsts so that a change to the constant is visible here
	k8sConfigGatewayContainerName = "gateway"
)

func newK8sConfigClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

// gatewayCollectorDeployment mirrors the shape of the deployment
// autoscaler/controllers/clustercollector hands to the configers: the gateway container is not the
// first one, and neither volumes nor volume mounts are set yet.
func gatewayCollectorDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-gateway",
			Namespace: k8sConfigNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: k8sConfigSidecarName},
						{Name: k8sConfigGatewayContainerName},
					},
				},
			},
		},
	}
}

func containerNamed(t *testing.T, dep *appsv1.Deployment, name string) corev1.Container {
	t.Helper()

	for _, container := range dep.Spec.Template.Spec.Containers {
		if container.Name == name {
			return container
		}
	}
	require.FailNowf(t, "container not found", "no container named %q in the deployment", name)

	return corev1.Container{}
}

// k8sConfigDestination builds a Destination CR, which is the production implementation of
// K8sExporterConfigurer. An empty secretName leaves the secret ref nil.
func k8sConfigDestination(id string, destType common.DestinationType, secretName string, data map[string]string) odigosv1.Destination {
	dest := odigosv1.Destination{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: k8sConfigNamespace,
		},
		Spec: odigosv1.DestinationSpec{
			Type:    destType,
			Data:    data,
			Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
		},
	}
	if secretName != "" {
		dest.Spec.SecretRef = &corev1.LocalObjectReference{Name: secretName}
	}

	return dest
}

func k8sConfigSecret(name, namespace string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       data,
	}
}

// caCertificatePEM returns a valid self-signed CA certificate in PEM form. The signing key is
// derived from a fixed seed so the certificate is the same on every run.
func caCertificatePEM(t *testing.T) []byte {
	t.Helper()

	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x2a}, ed25519.SeedSize))
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "odigos-clickhouse-test-ca"},
		NotBefore:             time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:              time.Date(2050, time.January, 1, 0, 0, 0, 0, time.UTC),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
