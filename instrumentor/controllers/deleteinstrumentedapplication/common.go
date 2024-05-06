package deleteinstrumentedapplication

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func reconcileWorkloadObject(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	logger := log.FromContext(ctx)
	instEffectiveEnabled, err := isWorkloadInstrumentationEffectiveEnabled(ctx, kubeClient, workloadObject)
	if err != nil {
		logger.Error(err, "error checking if instrumentation is effective")
		return err
	}

	if instEffectiveEnabled {
		return nil
	}

	if err := deleteWorkloadInstrumentedApplication(ctx, kubeClient, workloadObject); err != nil {
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

func deleteWorkloadInstrumentedApplication(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {

	ns := workloadObject.GetNamespace()
	name := workloadObject.GetName()
	kind := workloadObject.GetObjectKind().GroupVersionKind().Kind
	instrumentedApplicationName := utils.GetRuntimeObjectName(name, kind)

	err := kubeClient.Delete(ctx, &odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      instrumentedApplicationName,
		},
	})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	logger := log.FromContext(ctx)
	logger.V(1).Info("instrumented application deleted", "namespace", ns, "name", name, "kind", kind)
	return nil
}

func isWorkloadInstrumentationEffectiveEnabled(ctx context.Context, kubeClient client.Client, obj client.Object) (bool, error) {

	// if the object itself is labeled, we will use that value
	workloadLabels := obj.GetLabels()
	if val, exists := workloadLabels[consts.OdigosInstrumentationLabel]; exists {
		return val == consts.InstrumentationEnabled, nil
	}

	// we will get here if the workload instrumentation label is not set.
	// no label means inherit the instrumentation value from namespace.
	var ns corev1.Namespace
	err := kubeClient.Get(ctx, client.ObjectKey{Name: obj.GetNamespace()}, &ns)
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Error(err, "error fetching namespace object")
		return false, err
	}

	return isInstrumentationLabelEnabled(&ns), nil
}

func isInstrumentationLabelEnabled(workloadObject client.Object) bool {
	labels := workloadObject.GetLabels()
	return labels[consts.OdigosInstrumentationLabel] == consts.InstrumentationEnabled
}

func removeReportedNameAnnotation(ctx context.Context, kubeClient client.Client, workloadObject client.Object) error {
	if _, exists := workloadObject.GetAnnotations()[consts.OdigosReportedNameAnnotation]; !exists {
		return nil
	}

	return kubeClient.Patch(ctx, workloadObject, client.RawPatch(types.MergePatchType, []byte(`{"metadata":{"annotations":{"`+consts.OdigosReportedNameAnnotation+`":null}}}`)))
}
