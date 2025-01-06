package deleteinstrumentationconfig

import (
	"context"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func reconcileWorkloadObject(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	logger := log.FromContext(ctx)
	instEffectiveEnabled, err := workload.IsWorkloadInstrumentationEffectiveEnabled(ctx, kubeClient, workloadObject)
	if err != nil {
		logger.Error(err, "error checking if instrumentation is effective")
		return err
	}

	if instEffectiveEnabled {
		// Check if a Source object exists for this workload
		sourceList, err := v1alpha1.GetSourceListForWorkload(ctx, kubeClient, workloadObject)
		if err != nil {
			return err
		}
		if len(sourceList.Items) == 0 {
			return nil
		}
	}

	if err := deleteWorkloadInstrumentationConfig(ctx, kubeClient, workloadObject); err != nil {
		logger.Error(err, "error removing runtime details")
		return err
	}
	err = removeReportedNameAnnotation(ctx, kubeClient, workloadObject)
	if err != nil {
		logger.Error(err, "error removing reported name annotation ")
		return err
	}

	return nil
}

func deleteWorkloadInstrumentationConfig(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	ns := workloadObject.GetNamespace()
	name := workloadObject.GetName()
	kind := workload.WorkloadKindFromClientObject(workloadObject)
	instrumentedApplicationName := workload.CalculateWorkloadRuntimeObjectName(name, kind)

	instConfigErr := kubeClient.Delete(ctx, &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      instrumentedApplicationName,
		},
	})

	if instConfigErr != nil {
		return client.IgnoreNotFound(instConfigErr)
	}

	logger := log.FromContext(ctx)
	logger.V(1).Info("instrumented application deleted", "namespace", ns, "name", name, "kind", kind)
	return nil
}

func removeReportedNameAnnotation(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	if _, exists := workloadObject.GetAnnotations()[consts.OdigosReportedNameAnnotation]; !exists {
		return nil
	}

	return kubeClient.Patch(ctx, workloadObject, client.RawPatch(types.MergePatchType, []byte(`{"metadata":{"annotations":{"`+consts.OdigosReportedNameAnnotation+`":null}}}`)))
}
