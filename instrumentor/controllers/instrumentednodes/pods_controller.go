package instrumentednodes

import (
	"context"
	"fmt"
	"sync"
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

const maxPodNodeTrackerSize = 10000

// podNodeNamePredicate triggers on pod create/delete and when Spec.NodeName changes.
type podNodeNamePredicate struct{}

func (p podNodeNamePredicate) Create(e event.CreateEvent) bool {
	return true
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
	return true
}

func (p podNodeNamePredicate) Generic(e event.GenericEvent) bool {
	return false
}

// podNodeTracker maps pod name+namespace to its node so deletes can still sync the node.
// when a pod is deleted, we cannot "Get" it and discover the node it was running on,
// thus we need to maintain our track to be able to sync the node when a pod is deleted
type podNodeTracker struct {
	mu       sync.Mutex
	podNodes map[ctrl.Request]string
}

func newPodNodeTracker() *podNodeTracker {
	return &podNodeTracker{
		podNodes: make(map[ctrl.Request]string),
	}
}

func (t *podNodeTracker) Get(req ctrl.Request) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	node, ok := t.podNodes[req]
	return node, ok
}

func (t *podNodeTracker) Set(req ctrl.Request, nodeName string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.podNodes) >= maxPodNodeTrackerSize {
		return fmt.Errorf("pod node tracker is at max size: %d, skipping set", maxPodNodeTrackerSize)
	}
	t.podNodes[req] = nodeName
	return nil
}

func (t *podNodeTracker) Delete(req ctrl.Request) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.podNodes, req)
}

// PodsReconciler syncs the instrumented-pods node label when pods are created,
// scheduled, or deleted.
type PodsReconciler struct {
	client.Client
	PodNodes           *podNodeTracker
	NodeLabelRetention time.Duration
}

func (r *PodsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)

	var pod corev1.Pod
	err := r.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.handleDeletedPod(ctx, req)
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

	previousNode, _ := r.PodNodes.Get(req)
	nodeName := pod.Spec.NodeName
	if nodeName != "" {
		if err := r.PodNodes.Set(req, nodeName); err != nil {
			logger.Error(err, "error setting pod to node mapping", "pod", req.NamespacedName)
		}
	}

	var requeueAfter time.Duration
	if nodeName != "" {
		ra, err := syncNode(ctx, r.Client, nodeName, r.NodeLabelRetention)
		if err != nil {
			return utils.K8SUpdateErrorHandler(err)
		}
		requeueAfter = ra
		logger.Info("synced instrumented pods node label", "node", nodeName, "pod", req.NamespacedName)
	}

	if previousNode != "" && previousNode != nodeName {
		// node name should not change after the pod is scheduled,
		// so this case is not actually expected to happen,
		// but it's here for completeness, robustness practices,
		// so if for any reason it does, the controller keeps the states consistent
		logger.Info("pod node name changed after scheduling, syncing previous node", "pod", req.NamespacedName, "previousNode", previousNode, "nodeName", nodeName)
		ra, err := syncNode(ctx, r.Client, previousNode, r.NodeLabelRetention)
		if err != nil {
			return utils.K8SUpdateErrorHandler(err)
		}
		if ra > 0 && (requeueAfter == 0 || ra < requeueAfter) {
			requeueAfter = ra
		}
	}

	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	return ctrl.Result{}, nil
}

func (r *PodsReconciler) handleDeletedPod(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// deleted pod cannot be used to find it's node name, so we use our cache to sync the node
	// we need to do it for all pods (both with and without the label)
	// to correctly account for when a "no restart" pod is deleted and the label should be removed
	nodeName, ok := r.PodNodes.Get(req)
	if !ok || nodeName == "" {
		return ctrl.Result{}, nil
	}

	// shortcut - deleted pods can only remove the label, so if it's not here already we can return early
	n := &corev1.Node{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: nodeName}, n)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			r.PodNodes.Delete(req)
			return ctrl.Result{}, nil
		}
		return utils.K8SUpdateErrorHandler(err)
	}
	_, nodeHasInstrumentedPodsLabel := n.Labels[k8sconsts.FirstInstrumentedPodAtNodeLabel]
	if !nodeHasInstrumentedPodsLabel {
		// already has no label, so nothing to do
		r.PodNodes.Delete(req)
		return ctrl.Result{}, nil
	}

	requeueAfter, err := syncNode(ctx, r.Client, nodeName, r.NodeLabelRetention)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}
	if requeueAfter > 0 {
		// keep the tracker entry so a requeue can still resolve the node name
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// delete only after sync, since we need this record in cache if we are going to retry
	r.PodNodes.Delete(req)
	return ctrl.Result{}, nil
}
