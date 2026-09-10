package pro

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func TestEnterpriseRegistryPullSecretLabels(t *testing.T) {
	t.Parallel()

	labels := EnterpriseRegistryPullSecretLabels()
	if labels[k8sconsts.OdigosSystemLabelKey] != k8sconsts.OdigosSystemLabelValue {
		t.Fatalf("expected system-object label for pull secret")
	}
	if _, ok := labels[k8sconsts.OdigosSystemLabelCentralKey]; ok {
		t.Fatalf("did not expect central-system-object label for odigos namespace secret")
	}
}

func TestCopyImagePullSecretsIfMissing(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}

	enterprise, err := NewEnterpriseRegistryPullSecret("odigos-system", "test-token")
	if err != nil {
		t.Fatalf("failed to build enterprise secret: %v", err)
	}
	mirror := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mirror-pull",
			Namespace: "odigos-system",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte("mirror-auth")},
	}

	t.Run("no-op when sources missing", func(t *testing.T) {
		t.Parallel()
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		if err := CopyImagePullSecretsIfMissing(context.Background(), c, c, "odigos-system", "app", []string{"mirror-pull"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("same namespace is a no-op", func(t *testing.T) {
		t.Parallel()
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(enterprise.DeepCopy()).Build()
		if err := CopyImagePullSecretsIfMissing(context.Background(), c, c, "odigos-system", "odigos-system", []string{k8sconsts.OdigosEnterpriseRegistryPullSecretName}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("copies configured secrets with system-object label", func(t *testing.T) {
		t.Parallel()
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(enterprise.DeepCopy(), mirror.DeepCopy()).Build()
		names := []string{"mirror-pull", k8sconsts.OdigosEnterpriseRegistryPullSecretName}
		if err := CopyImagePullSecretsIfMissing(context.Background(), c, c, "odigos-system", "app", names); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, name := range names {
			var dest corev1.Secret
			if err := c.Get(context.Background(), client.ObjectKey{Namespace: "app", Name: name}, &dest); err != nil {
				t.Fatalf("expected copied secret %q: %v", name, err)
			}
			if dest.Labels[k8sconsts.OdigosSystemLabelKey] != k8sconsts.OdigosSystemLabelValue {
				t.Fatalf("expected system-object label on copied secret %q", name)
			}
		}
	})

	t.Run("leaves existing dest secret in place", func(t *testing.T) {
		t.Parallel()
		existing := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mirror-pull",
				Namespace: "app",
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte("existing")},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(mirror.DeepCopy(), existing).Build()
		if err := CopyImagePullSecretsIfMissing(context.Background(), c, c, "odigos-system", "app", []string{"mirror-pull"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var dest corev1.Secret
		if err := c.Get(context.Background(), client.ObjectKey{Namespace: "app", Name: "mirror-pull"}, &dest); err != nil {
			t.Fatalf("expected dest secret: %v", err)
		}
		if string(dest.Data[corev1.DockerConfigJsonKey]) != "existing" {
			t.Fatalf("existing secret was overwritten")
		}
	})
}
