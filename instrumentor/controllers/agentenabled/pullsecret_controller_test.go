package agentenabled

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

func TestSyncOdigosPullSecretsAnnotationPredicate(t *testing.T) {
	t.Parallel()

	p := syncOdigosPullSecretsAnnotationPredicate{}
	secret := func(value string) *corev1.Secret {
		s := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "odigos-enterprise-registry"}}
		if value != "" {
			s.Annotations = map[string]string{k8sconsts.SyncOdigosPullSecretsAnnotation: value}
		}
		return s
	}

	if !p.Create(event.CreateEvent{Object: secret("1")}) {
		t.Fatalf("expected create with annotation to match")
	}
	if p.Create(event.CreateEvent{Object: secret("")}) {
		t.Fatalf("expected create without annotation to be ignored")
	}
	if !p.Update(event.UpdateEvent{ObjectOld: secret("1"), ObjectNew: secret("2")}) {
		t.Fatalf("expected annotation value change to match")
	}
	if p.Update(event.UpdateEvent{ObjectOld: secret("1"), ObjectNew: secret("1")}) {
		t.Fatalf("expected unchanged annotation to be ignored")
	}
}
