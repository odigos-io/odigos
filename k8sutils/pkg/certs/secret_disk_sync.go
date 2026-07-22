package certs

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultCertName = "tls.crt"
	defaultKeyName  = "tls.key"
)

// WebhookCertDir is a writable directory for webhook serving certs.
// It is intentionally separate from any Secret volume mount so readiness does
// not wait on kubelet secret projection delays.
func WebhookCertDir() string {
	return filepath.Join(os.TempDir(), "odigos-webhook-certs")
}

// SecretDiskSync copies TLS material from a Secret into CertDir.
// cert-controller writes certs to the Secret; this runnable materializes them
// on disk for the webhook server without waiting for kubelet mounts.
type SecretDiskSync struct {
	Client   client.Client
	Secret   types.NamespacedName
	CertDir  string
	CertName string
	KeyName  string
}

func (s *SecretDiskSync) NeedLeaderElection() bool {
	return false
}

func (s *SecretDiskSync) Start(ctx context.Context) error {
	certName := s.CertName
	if certName == "" {
		certName = defaultCertName
	}
	keyName := s.KeyName
	if keyName == "" {
		keyName = defaultKeyName
	}
	certDir := s.CertDir
	if certDir == "" {
		certDir = WebhookCertDir()
	}

	var lastCert, lastKey []byte
	err := wait.PollUntilContextCancel(ctx, 200*time.Millisecond, true, func(ctx context.Context) (bool, error) {
		secret := &corev1.Secret{}
		if err := s.Client.Get(ctx, s.Secret, secret); err != nil {
			return false, nil
		}
		cert := secret.Data[certName]
		key := secret.Data[keyName]
		if len(cert) == 0 || len(key) == 0 {
			return false, nil
		}
		if bytes.Equal(cert, lastCert) && bytes.Equal(key, lastKey) {
			return false, nil
		}
		if err := os.MkdirAll(certDir, 0o700); err != nil {
			return false, nil
		}
		if err := os.WriteFile(filepath.Join(certDir, certName), cert, 0o600); err != nil {
			return false, nil
		}
		if err := os.WriteFile(filepath.Join(certDir, keyName), key, 0o600); err != nil {
			return false, nil
		}
		lastCert = bytes.Clone(cert)
		lastKey = bytes.Clone(key)
		return false, nil
	})
	if ctx.Err() != nil {
		return nil
	}
	return err
}
