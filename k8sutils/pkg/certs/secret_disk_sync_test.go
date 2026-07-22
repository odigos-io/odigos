package certs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSecretDiskSyncWritesCertFiles(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-cert",
			Namespace: "odigos-test",
		},
		Data: map[string][]byte{
			defaultCertName: []byte("cert-data"),
			defaultKeyName:  []byte("key-data"),
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	certDir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	syncer := &SecretDiskSync{
		Client:  c,
		Secret:  types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace},
		CertDir: certDir,
	}
	done := make(chan error, 1)
	go func() {
		done <- syncer.Start(ctx)
	}()

	require.Eventually(t, func() bool {
		cert, err := os.ReadFile(filepath.Join(certDir, defaultCertName))
		if err != nil {
			return false
		}
		key, err := os.ReadFile(filepath.Join(certDir, defaultKeyName))
		if err != nil {
			return false
		}
		return string(cert) == "cert-data" && string(key) == "key-data"
	}, 2*time.Second, 20*time.Millisecond)

	cancel()
	require.NoError(t, <-done)
}
