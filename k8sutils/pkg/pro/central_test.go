package pro

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

func TestGetCentralOnPremToken(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosCentralSecretName,
			Namespace: "odigos-central",
		},
		Data: map[string][]byte{
			k8sconsts.OdigosOnpremTokenEnvName: []byte("central-token"),
		},
	})

	token, err := GetCentralOnPremToken(context.Background(), client, "odigos-central")
	if err != nil {
		t.Fatalf("GetCentralOnPremToken() error = %v", err)
	}
	if token != "central-token" {
		t.Fatalf("GetCentralOnPremToken() = %q, want central-token", token)
	}
}

func TestGetCentralOnPremTokenLegacyKey(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosCentralSecretName,
			Namespace: "odigos-central",
		},
		Data: map[string][]byte{
			k8sconsts.OdigosOnpremTokenSecretKey: []byte("legacy-token"),
		},
	})

	token, err := GetCentralOnPremToken(context.Background(), client, "odigos-central")
	if err != nil {
		t.Fatalf("GetCentralOnPremToken() error = %v", err)
	}
	if token != "legacy-token" {
		t.Fatalf("GetCentralOnPremToken() = %q, want legacy-token", token)
	}
}

func TestGetCentralOnPremTokenMissingSecret(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	_, err := GetCentralOnPremToken(context.Background(), client, "odigos-central")
	if err == nil {
		t.Fatal("GetCentralOnPremToken() expected error for missing secret")
	}
}

func TestUsesOdigosEnterpriseRegistry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config common.OdigosConfiguration
		want   bool
	}{
		{
			name:   "default",
			config: common.OdigosConfiguration{},
			want:   true,
		},
		{
			name:   "openshift",
			config: common.OdigosConfiguration{OpenshiftEnabled: true},
			want:   false,
		},
		{
			name:   "custom prefix",
			config: common.OdigosConfiguration{ImagePrefix: "myregistry.io/odigos"},
			want:   false,
		},
		{
			name:   "explicit enterprise prefix",
			config: common.OdigosConfiguration{ImagePrefix: k8sconsts.OdigosEnterpriseImagePrefix},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := UsesOdigosEnterpriseRegistry(tt.config); got != tt.want {
				t.Fatalf("UsesOdigosEnterpriseRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}
