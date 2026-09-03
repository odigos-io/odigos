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

func TestImagePullSecretNamesToCopy(t *testing.T) {
	t.Parallel()

	names := ImagePullSecretNamesToCopy([]string{"mirror-pull", "", "mirror-pull", k8sconsts.OdigosEnterpriseRegistryPullSecretName})
	if len(names) != 2 {
		t.Fatalf("expected 2 unique names, got %#v", names)
	}
	if names[0] != "mirror-pull" || names[1] != k8sconsts.OdigosEnterpriseRegistryPullSecretName {
		t.Fatalf("unexpected name order %#v", names)
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
		present, err := CopyImagePullSecretsIfMissing(context.Background(), c, c, "odigos-system", "app", []string{"mirror-pull"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(present) != 0 {
			t.Fatalf("expected no secrets to be present, got %#v", present)
		}
	})

	t.Run("same namespace is present without copying", func(t *testing.T) {
		t.Parallel()
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(enterprise.DeepCopy()).Build()
		present, err := CopyImagePullSecretsIfMissing(context.Background(), c, c, "odigos-system", "odigos-system", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(present) != 1 || present[0] != k8sconsts.OdigosEnterpriseRegistryPullSecretName {
			t.Fatalf("expected enterprise secret in source namespace, got %#v", present)
		}
	})

	t.Run("copies configured and enterprise secrets with system-object label", func(t *testing.T) {
		t.Parallel()
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(enterprise.DeepCopy(), mirror.DeepCopy()).Build()
		present, err := CopyImagePullSecretsIfMissing(context.Background(), c, c, "odigos-system", "app", []string{"mirror-pull"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(present) != 2 {
			t.Fatalf("expected both secrets copied, got %#v", present)
		}

		for _, name := range []string{"mirror-pull", k8sconsts.OdigosEnterpriseRegistryPullSecretName} {
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
		present, err := CopyImagePullSecretsIfMissing(context.Background(), c, c, "odigos-system", "app", []string{"mirror-pull"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(present) != 1 || present[0] != "mirror-pull" {
			t.Fatalf("expected existing secret to count as present, got %#v", present)
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
