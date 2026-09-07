package instrumentednodes

import (
	"context"
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

const podNodeNameIndex = "spec.nodeName"

func syncNode(ctx context.Context, c client.Client, nodeName string, nodeLabelRetention time.Duration) (time.Duration, error) {
	var node corev1.Node
	err := c.Get(ctx, client.ObjectKey{Name: nodeName}, &node)
	if err != nil {
		return 0, client.IgnoreNotFound(err)
	}

	hasInstrumented, err := hasInstrumentedPod(ctx, c, node.Name)
	if err != nil {
		return 0, err
	}

	return syncFirstInstrumentedPodAtNodeLabel(ctx, c, &node, hasInstrumented, nodeLabelRetention)
}

func hasInstrumentedPod(ctx context.Context, c client.Client, nodeName string) (bool, error) {
	var pods corev1.PodList
	err := c.List(ctx, &pods, client.MatchingFields{podNodeNameIndex: nodeName})
	if err != nil {
		return false, err
	}

	// first do a quick and cheap check for the OdigosAgentsMetaHashLabel
	// if found, we don't need to check for pods instrumented with "no restart"
	for i := range pods.Items {
		pod := &pods.Items[i]
		if _, ok := pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel]; ok {
			return true, nil
		}
	}

	// this check requires us to list the instrumentation configs,
	// so we do it only when neccecary
	for i := range pods.Items {
		pod := &pods.Items[i]
		hasIC, err := podHasInstrumentationConfig(ctx, c, pod)
		if err != nil {
			return false, err
		}
		if hasIC {
			return true, nil
		}
	}

	return false, nil
}

func podHasInstrumentationConfig(ctx context.Context, c client.Client, pod *corev1.Pod) (bool, error) {
	pw, err := workload.PodWorkloadObject(pod)
	if err != nil || pw == nil {
		return false, err
	}

	icName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	var ic odigosv1.InstrumentationConfig
	err = c.Get(ctx, client.ObjectKey{Namespace: pw.Namespace, Name: icName}, &ic)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func syncFirstInstrumentedPodAtNodeLabel(ctx context.Context, c client.Client, node *corev1.Node, hasInstrumented bool, nodeLabelRetention time.Duration) (time.Duration, error) {
	labelValue, hasLabel := node.Labels[k8sconsts.FirstInstrumentedPodAtNodeLabel]

	// label already reflects the desired state
	if hasInstrumented == hasLabel {
		return 0, nil
	}

	if hasInstrumented {
		// node just got it's first instrumented pod: sttamp it with the current time
		now := time.Now().UTC().Format(k8sconsts.FirstInstrumentedPodAtNodeLabelTimeFormat)
		err := patchFirstInstrumentedPodAtNodeLabel(ctx, c, node, now)
		return 0, err
	} else {
		// no instrumented pods left: remove the label but only after the retention period
		requeueAfter, delayRemoval := retentionRemaining(labelValue, nodeLabelRetention)
		if delayRemoval {
			return requeueAfter, nil
		}
		err := patchFirstInstrumentedPodAtNodeLabel(ctx, c, node, nil)
		return 0, err
	}
}

func patchFirstInstrumentedPodAtNodeLabel(ctx context.Context, c client.Client, node *corev1.Node, value any) error {
	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"labels": map[string]any{
				k8sconsts.FirstInstrumentedPodAtNodeLabel: value,
			},
		},
	})
	if err != nil {
		return err
	}
	return c.Patch(ctx, node, client.RawPatch(types.MergePatchType, patch))
}

// retentionRemaining reports whether removal should wait, and for how long.
// Unparseable label values (e.g. legacy "true") are treated as eligible for immediate removal.
func retentionRemaining(labelValue string, nodeLabelRetention time.Duration) (time.Duration, bool) {
	if nodeLabelRetention <= 0 {
		return 0, false
	}
	discoveredAt, err := time.Parse(k8sconsts.FirstInstrumentedPodAtNodeLabelTimeFormat, labelValue)
	if err != nil {
		return 0, false
	}
	remaining := discoveredAt.UTC().Add(nodeLabelRetention).Sub(time.Now().UTC())
	if remaining <= 0 {
		return 0, false
	}
	return remaining, true
}
