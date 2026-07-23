package certs

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var lastCert, lastKey []byte
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			cert, key, ok := s.readTLSMaterial(ctx, certName, keyName)
			if !ok {
				continue
			}
			if bytes.Equal(cert, lastCert) && bytes.Equal(key, lastKey) {
				continue
			}
			if err := writeTLSMaterial(certDir, certName, keyName, cert, key); err != nil {
				continue
			}
			lastCert = bytes.Clone(cert)
			lastKey = bytes.Clone(key)
		}
	}
}

func (s *SecretDiskSync) readTLSMaterial(ctx context.Context, certName, keyName string) (cert, key []byte, ok bool) {
	secret := &corev1.Secret{}
	if err := s.Client.Get(ctx, s.Secret, secret); err != nil {
		return nil, nil, false
	}
	cert = secret.Data[certName]
	key = secret.Data[keyName]
	if len(cert) == 0 || len(key) == 0 {
		return nil, nil, false
	}
	return cert, key, true
}

func writeTLSMaterial(certDir, certName, keyName string, cert, key []byte) error {
	if err := os.MkdirAll(certDir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(certDir, certName), cert, 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(certDir, keyName), key, 0o600)
}
