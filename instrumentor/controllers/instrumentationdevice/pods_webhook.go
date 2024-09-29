package instrumentationdevice

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
)

type PodsWebhook struct{}

var _ webhook.CustomDefaulter = &PodsWebhook{}

func (p *PodsWebhook) Default(ctx context.Context, obj runtime.Object) error {
	// TODO(edenfed): add object selector to mutatingwebhookconfiguration
	log := logf.FromContext(ctx)
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	pod.Annotations["odigos.io/instrumented-webhook"] = "true"
	log.V(0).Info("Defaulted Pod", "name", pod.Name)
	return nil
}
