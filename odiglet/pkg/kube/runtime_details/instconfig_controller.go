package runtime_details

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// replaced by odiglet/pkg/kube/runtime_details/instrumentationconfigs_controller.go
// which does not rely on the RuntimeDetailsInvalidated flag
// left here until the migration is complete
// Deprecated: the new runtime inspection logic is found in odiglet/pkg/kube/runtime_details/instrumentationconfigs_controller.go
type DeprecatedInstrumentationConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (i *DeprecatedInstrumentationConfigReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)
	var instConfig odigosv1.InstrumentationConfig
	err := i.Get(ctx, request.NamespacedName, &instConfig)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to get InstrumentationConfig")
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// This reconciler is only interested in InstrumentationConfig objects that have their RuntimeDetailsInvalidated field set to true
	if !instConfig.Spec.RuntimeDetailsInvalidated {
		return reconcile.Result{}, nil
	}

	if len(instConfig.OwnerReferences) != 1 {
		return reconcile.Result{}, fmt.Errorf("InstrumentationConfig %s/%s has %d owner references, expected 1", instConfig.Namespace, instConfig.Name, len(instConfig.OwnerReferences))
	}

	workload, labels, err := getWorkloadAndLabelsfromOwner(ctx, i.Client, instConfig.Namespace, instConfig.OwnerReferences[0])
	if err != nil {
		logger.Error(err, "Failed to get workload and labels from owner")
		return reconcile.Result{}, err
	}
	err = inspectRuntimesOfRunningPods(ctx, &logger, labels, i.Client, i.Scheme, workload)
	if err != nil {
		return reconcile.Result{}, ignoreNoPodsFoundError(err)
	}

	// Patch RuntimeDetailsInvalidated to false after runtime details have been recalculated
	updated := instConfig.DeepCopy()
	updated.Spec.RuntimeDetailsInvalidated = false
	patch := client.MergeFrom(&instConfig)
	err = i.Patch(ctx, updated, patch)
	return reconcile.Result{}, err
}

func getWorkloadAndLabelsfromOwner(ctx context.Context, k8sClient client.Client, ns string, ownerReference metav1.OwnerReference) (client.Object, map[string]string, error) {
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(ownerReference)
	if err != nil {
		return nil, nil, err
	}

	switch workloadKind {
	case "Deployment":
		var dep appsv1.Deployment
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: workloadName}, &dep)
		if err != nil {
			return nil, nil, err
		}
		return &dep, dep.Spec.Selector.MatchLabels, nil
	case "DaemonSet":
		var ds appsv1.DaemonSet
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: workloadName}, &ds)
		if err != nil {
			return nil, nil, err
		}

		return &ds, ds.Spec.Selector.MatchLabels, nil
	case "StatefulSet":
		var sts appsv1.StatefulSet
		err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: workloadName}, &sts)
		if err != nil {
			return nil, nil, err
		}

		return &sts, sts.Spec.Selector.MatchLabels, nil
	}

	return nil, nil, errors.New("workload kind not supported")
}
