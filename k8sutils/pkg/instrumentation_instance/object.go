package instrumentation_instance

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func DeleteInstrumentationInstance(ctx context.Context, owner client.Object, containerName string, kubeClient client.Client, pid int) error {
	instrumentationInstanceName := InstrumentationInstanceName(owner.GetName(), pid)
	err := kubeClient.Delete(ctx, &v1alpha1.InstrumentationInstance{
		ObjectMeta: v1.ObjectMeta{
			Name:      instrumentationInstanceName,
			Namespace: owner.GetNamespace(),
		},
	})
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}
