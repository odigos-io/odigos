package instrumentednodes

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

// nodeInstrumentedPodsLabelPredicate triggers on node create and when
// FirstInstrumentedPodAtNodeLabel is added, removed, or changed.
type nodeInstrumentedPodsLabelPredicate struct{}

func (p nodeInstrumentedPodsLabelPredicate) Create(e event.CreateEvent) bool {
	return true
}

func (p nodeInstrumentedPodsLabelPredicate) Update(e event.UpdateEvent) bool {
	oldNode, okOld := e.ObjectOld.(*corev1.Node)
	newNode, okNew := e.ObjectNew.(*corev1.Node)
	if !okOld || !okNew {
		return false
	}
	return oldNode.Labels[k8sconsts.FirstInstrumentedPodAtNodeLabel] != newNode.Labels[k8sconsts.FirstInstrumentedPodAtNodeLabel]
}

func (p nodeInstrumentedPodsLabelPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p nodeInstrumentedPodsLabelPredicate) Generic(e event.GenericEvent) bool {
	return false
}

type NodesReconciler struct {
	client.Client
	NodeLabelRetention time.Duration
}

func (r *NodesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	requeueAfter, err := syncNode(ctx, r.Client, req.Name, r.NodeLabelRetention)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}
	// Periodically re-sync so the label is cleared after instrumented pods are
	// deleted. The pods controller cannot handle deletes (node name is unknown
	// once the pod is gone), so this is the path that eventually removes the label.
	// it means new pods are handled right away, but pods being deleted are handled at most 1 minute after they are deleted.
	const periodicRequeue = time.Minute
	if requeueAfter == 0 || requeueAfter > periodicRequeue {
		requeueAfter = periodicRequeue
	}
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}
