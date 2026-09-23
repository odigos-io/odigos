package agentenabled

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/version"

	"github.com/odigos-io/odigos/api/k8sconsts"
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

func TestShouldInjectInitContainerPullSecrets(t *testing.T) {
	t.Parallel()

	if shouldInjectInitContainerPullSecrets(nil) {
		t.Fatal("expected no inject when cluster version is unknown")
	}
	if shouldInjectInitContainerPullSecrets(version.MustParse("1.34.0")) {
		t.Fatal("expected no inject on Kubernetes 1.34")
	}
	if !shouldInjectInitContainerPullSecrets(version.MustParse("1.35.0")) {
		t.Fatal("expected inject on Kubernetes 1.35")
	}
	if !shouldInjectInitContainerPullSecrets(version.MustParse("1.36.1")) {
		t.Fatal("expected inject on Kubernetes 1.36")
	}
}
