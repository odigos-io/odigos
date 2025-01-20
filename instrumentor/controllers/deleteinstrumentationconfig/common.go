package deleteinstrumentationconfig

import (
	"context"

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
		return nil
	}

	// Check if a Source object exists for this workload
	sourceList, err := odigosv1.GetSources(ctx, kubeClient, workloadObject)
	if err != nil {
		return err
	}
	if sourceList.Workload != nil || sourceList.Namespace != nil {
		// Note that if a Source is being deleted, but still has the finalizer, it will still show up in this List
		// So we can't use this check to trigger un-instrumentation via Source deletion
		return nil
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
	logger := log.FromContext(ctx)
	ns := workloadObject.GetNamespace()
	name := workloadObject.GetName()
	kind := workload.WorkloadKindFromClientObject(workloadObject)
	instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
	logger.V(1).Info("deleting instrumentationconfig", "name", instrumentationConfigName, "namespace", ns)

	instConfigErr := kubeClient.Delete(ctx, &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      instrumentationConfigName,
		},
	})

	if instConfigErr != nil {
		return client.IgnoreNotFound(instConfigErr)
	}

	return nil
}

func removeReportedNameAnnotation(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	if _, exists := workloadObject.GetAnnotations()[consts.OdigosReportedNameAnnotation]; !exists {
		return nil
	}

	return kubeClient.Patch(ctx, workloadObject, client.RawPatch(types.MergePatchType, []byte(`{"metadata":{"annotations":{"`+consts.OdigosReportedNameAnnotation+`":null}}}`)))
}
