package agentenabled

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"
)

func TestInjectImagePullSecret(t *testing.T) {
	t.Parallel()

	pod := &corev1.Pod{}
	injectImagePullSecret(pod, k8sconsts.OdigosEnterpriseRegistryPullSecretName)
	if len(pod.Spec.ImagePullSecrets) != 1 || pod.Spec.ImagePullSecrets[0].Name != k8sconsts.OdigosEnterpriseRegistryPullSecretName {
		t.Fatalf("expected enterprise pull secret to be injected, got %#v", pod.Spec.ImagePullSecrets)
	}

	injectImagePullSecret(pod, k8sconsts.OdigosEnterpriseRegistryPullSecretName)
	if len(pod.Spec.ImagePullSecrets) != 1 {
		t.Fatalf("expected inject to be idempotent, got %#v", pod.Spec.ImagePullSecrets)
	}

	pod.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "other"}}
	injectImagePullSecret(pod, k8sconsts.OdigosEnterpriseRegistryPullSecretName)
	if len(pod.Spec.ImagePullSecrets) != 2 {
		t.Fatalf("expected existing pull secrets to be preserved, got %#v", pod.Spec.ImagePullSecrets)
	}
}

func TestEnsureImagePullSecretsForPod(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add core scheme: %v", err)
	}

	enterprise, err := pro.NewEnterpriseRegistryPullSecret("odigos-system", "test-token")
	if err != nil {
		t.Fatalf("failed to build source secret: %v", err)
	}
	mirror := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mirror-pull",
			Namespace: "odigos-system",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: []byte("mirror-auth")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(enterprise, mirror).Build()
	pod := &corev1.Pod{}

	if err := ensureImagePullSecretsForPod(context.Background(), c, "odigos-system", "app", pod, []string{"mirror-pull"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pod.Spec.ImagePullSecrets) != 2 {
		t.Fatalf("expected both pull secrets on pod, got %#v", pod.Spec.ImagePullSecrets)
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
}
