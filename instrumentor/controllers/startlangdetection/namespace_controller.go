package startlangdetection

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
)

type NamespacesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (n *NamespacesReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var ns corev1.Namespace
	err := n.Get(ctx, request.NamespacedName, &ns)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := sourceutils.MigrateInstrumentationLabelToSource(ctx, n.Client, &ns, "Namespace"); err != nil {
		return ctrl.Result{}, err
	}

	enabled, err := sourceutils.IsObjectInstrumentedBySource(ctx, n.Client, &ns)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !enabled {
		return ctrl.Result{}, nil
	}

	logger.V(0).Info("Namespace enabled for instrumentation, recalculating runtime details of relevant workloads")
	return ctrl.Result{}, syncNamespaceWorkloads(ctx, n.Client, n.Scheme, ns.GetName())
}
