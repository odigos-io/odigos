package deleteinstrumentedapplication

import (
	"context"

	"github.com/go-logr/logr"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common/consts"
	"github.com/keyval-dev/odigos/common/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func removeRuntimeDetails(ctx context.Context, kubeClient client.Client, ns string, name string, kind string, logger logr.Logger) error {
	runtimeName := utils.GetRuntimeObjectName(name, kind)
	var runtimeDetails odigosv1.InstrumentedApplication
	err := kubeClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: runtimeName}, &runtimeDetails)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	err = kubeClient.Delete(ctx, &runtimeDetails)
	if err != nil {
		return err
	}

	logger.V(0).Info("removed runtime details due to label change")
	return nil
}

func isObjectInstrumentationEffectiveEnabled(logger logr.Logger, ctx context.Context, kubeClient client.Client, obj client.Object) (bool, error) {

	// if the object itself is labeled, we will use that value
	labels := obj.GetLabels()
	if labels != nil {
		val, exists := labels[consts.OdigosInstrumentationLabel]
		if exists {
			return val == consts.InstrumentationEnabled, nil
		}
	}

	// we will get here if the instrumentation label is not set.
	// in which case, we would want to check the namespace value
	var ns corev1.Namespace
	err := kubeClient.Get(ctx, client.ObjectKey{Name: obj.GetNamespace()}, &ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		logger.Error(err, "error fetching namespace object")
		return false, err
	}

	nsInstrumentationEnabled := isInstrumentationLabelEnabled(&ns)
	return nsInstrumentationEnabled, nil
}

func isInstrumentationLabelEnabled(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels != nil {
		val, exists := labels[consts.OdigosInstrumentationLabel]
		if exists && val == consts.InstrumentationEnabled {
			return true
		}
	}

	return false
}

func removeReportedNameAnnotation(obj client.Object) bool {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}

	if _, exists := annotations[consts.OdigosReportedNameAnnotation]; !exists {
		return false
	}

	delete(annotations, consts.OdigosReportedNameAnnotation)
	obj.SetAnnotations(annotations)
	return true
}
