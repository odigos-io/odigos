package instrumentednodes

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

// podNodeNamePredicate triggers on pod create and when Spec.NodeName changes.
// Deletes are ignored: once a pod is gone we cannot discover its node name.
type podNodeNamePredicate struct{}

func (p podNodeNamePredicate) Create(e event.CreateEvent) bool {
	pod, ok := e.Object.(*corev1.Pod)
	return ok && pod.Spec.NodeName != ""
}

func (p podNodeNamePredicate) Update(e event.UpdateEvent) bool {
	oldPod, okOld := e.ObjectOld.(*corev1.Pod)
	newPod, okNew := e.ObjectNew.(*corev1.Pod)
	if !okOld || !okNew {
		return false
	}
	// NodeName only transitions from empty to set when the pod is scheduled;
	// Kubernetes never reassigns an existing pod to a different node.
	return oldPod.Spec.NodeName != newPod.Spec.NodeName
}

func (p podNodeNamePredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (p podNodeNamePredicate) Generic(e event.GenericEvent) bool {
	return false
}

// PodsReconciler syncs the instrumented-pods node label when pods are created
// or scheduled. Deleted pods are not handled here (node name is unknown).
type PodsReconciler struct {
	client.Client
	NodeLabelRetention time.Duration
}

func (r *PodsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)

	var pod corev1.Pod
	err := r.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	_, hasOdigosAgentsMetaHashLabel := pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel]
	if !hasOdigosAgentsMetaHashLabel {
		// for live pods, the pod reconcile only cares about them if they have the label,
		// e.g. - to set the node label when the first pod with agents is scheduled on the node.
		// "new sources" and "no restart" cases are handled by different reconcilers
		return ctrl.Result{}, nil
	}

	nodeName := pod.Spec.NodeName
	if nodeName == "" {
		return ctrl.Result{}, nil
	}

	requeueAfter, err := syncNode(ctx, r.Client, nodeName, r.NodeLabelRetention)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}
	logger.Info("synced instrumented pods node label", "node", nodeName, "pod", req.NamespacedName)

	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	return ctrl.Result{}, nil
}
