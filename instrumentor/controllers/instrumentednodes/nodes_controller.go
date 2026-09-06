package instrumentednodes

import (
	"context"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

const podNodeNameIndex = "spec.nodeName"

// NodesReconciler keeps k8sconsts.InstrumentedPodsNodeLabel on each Node to
// indicate whether the node currently runs any instrumented pods.
type NodesReconciler struct {
	client.Client
}

func (r *NodesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)

	err := r.syncNode(ctx, req.Name)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}

	logger.Info("synced instrumented pods node label", "node", req.Name)
	return ctrl.Result{}, nil
}

func (r *NodesReconciler) syncNode(ctx context.Context, nodeName string) error {
	var node corev1.Node
	err := r.Get(ctx, client.ObjectKey{Name: nodeName}, &node)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	hasInstrumented, err := r.hasInstrumentedPod(ctx, node.Name)
	if err != nil {
		return err
	}

	return r.syncInstrumentedPodsNodeLabel(ctx, &node, hasInstrumented)
}

func (r *NodesReconciler) hasInstrumentedPod(ctx context.Context, nodeName string) (bool, error) {
	var pods corev1.PodList
	err := r.List(ctx, &pods, client.MatchingFields{podNodeNameIndex: nodeName})
	if err != nil {
		return false, err
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if _, ok := pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel]; ok {
			return true, nil
		}
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		hasIC, err := r.podHasInstrumentationConfig(ctx, pod)
		if err != nil {
			return false, err
		}
		if hasIC {
			return true, nil
		}
	}

	return false, nil
}

func (r *NodesReconciler) podHasInstrumentationConfig(ctx context.Context, pod *corev1.Pod) (bool, error) {
	pw, err := workload.PodWorkloadObject(pod)
	if err != nil || pw == nil {
		return false, err
	}

	icName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	var ic odigosv1.InstrumentationConfig
	err = r.Get(ctx, client.ObjectKey{Namespace: pw.Namespace, Name: icName}, &ic)
	if err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

func (r *NodesReconciler) syncInstrumentedPodsNodeLabel(ctx context.Context, node *corev1.Node, hasInstrumented bool) error {
	_, hasLabel := node.Labels[k8sconsts.InstrumentedPodsNodeLabel]
	if hasInstrumented == hasLabel {
		return nil
	}

	original := node.DeepCopy()
	if node.Labels == nil {
		node.Labels = map[string]string{}
	}
	if hasInstrumented {
		node.Labels[k8sconsts.InstrumentedPodsNodeLabel] = k8sconsts.InstrumentedPodsNodeLabelValue
	} else {
		delete(node.Labels, k8sconsts.InstrumentedPodsNodeLabel)
	}

	return r.Patch(ctx, node, client.MergeFrom(original))
}

// removeInstrumentedPodsNodeLabelsRunnable strips leftover InstrumentedPodsNodeLabel
// from all nodes when schedule-only-on-instrumented-nodes is disabled.
type removeInstrumentedPodsNodeLabelsRunnable struct {
	Client client.Client
}

func (r *removeInstrumentedPodsNodeLabelsRunnable) NeedLeaderElection() bool {
	return true
}

func (r *removeInstrumentedPodsNodeLabelsRunnable) Start(ctx context.Context) error {
	var nodes corev1.NodeList
	if err := r.Client.List(ctx, &nodes, client.HasLabels{k8sconsts.InstrumentedPodsNodeLabel}); err != nil {
		return err
	}

	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"labels": map[string]any{
				k8sconsts.InstrumentedPodsNodeLabel: nil,
			},
		},
	})
	if err != nil {
		return err
	}

	for i := range nodes.Items {
		if err := r.Client.Patch(ctx, &nodes.Items[i], client.RawPatch(types.MergePatchType, patch)); err != nil {
			return err
		}
	}
	return nil
}
