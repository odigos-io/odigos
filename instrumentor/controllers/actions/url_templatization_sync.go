package actions

import (
	"context"
	"encoding/json"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// URLTemplatizationSyncMode selects how SyncUrlTemplatizationProcessor reconciles the shared Processor CR.
type URLTemplatizationSyncMode int

const (
	// URLTemplatizationSyncCreateIfMissing (0): use from ActionReconcile only.
	// Ensures the shared Processor CR exists when missing, but if it already exists returns without patching.
	URLTemplatizationSyncCreateIfMissing URLTemplatizationSyncMode = 0
	// URLTemplatizationSyncApplyFull (1): use when the shared Processor must match
	// current Actions + node CollectorsGroup state.
	// Always builds and applies (or deletes when no URL-templatization actions).
	URLTemplatizationSyncApplyFull URLTemplatizationSyncMode = 1
)

func hasAnyUrlTemplatizationAction(ctx context.Context, c client.Client, namespace string) (bool, error) {
	var list odigosv1.ActionList
	// ToDo: implement mutating webhook to inject labels into url template actions
	// and then use the label as an identifier to only list URL template actions
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return false, err
	}
	for i := range list.Items {
		a := &list.Items[i]
		if a.Spec.URLTemplatization != nil && !a.Spec.Disabled {
			return true, nil
		}
	}
	return false, nil
}

func buildUrlTemplatizationProcessor(namespace string, spanMetricsEnabled bool) (*odigosv1.Processor, error) {
	cfg := actions.URLTemplatizationConfig{}
	configJSON, err := json.Marshal(map[string]interface{}{
		"odigos_config_extension": k8sconsts.OdigosConfigK8sExtensionType,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal url templatization processor config: %w", err)
	}
	// Span metrics are built in the node collector pipeline and label series using span name and
	// http.route etc. Without templating those values first, metrics explode in cardinality.
	// When span metrics are on, the shared processor must run on the node collector so routes are
	// normalized before metrics; otherwise gateway-only is enough for traces.
	collectorRoles := cfg.SharedProcessorCollectorRoles(spanMetricsEnabled)
	roles := make([]odigosv1.CollectorsGroupRole, 0, len(collectorRoles))
	for _, r := range collectorRoles {
		roles = append(roles, odigosv1.CollectorsGroupRole(r))
	}
	return &odigosv1.Processor{
		TypeMeta: metav1.TypeMeta{APIVersion: "odigos.io/v1alpha1", Kind: "Processor"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.URLTemplatizationProcessorName,
			Namespace: namespace,
		},
		Spec: odigosv1.ProcessorSpec{
			Type:            cfg.ProcessorType(),
			ProcessorName:   "URL Templatization",
			Disabled:        false,
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles:  roles,
			OrderHint:       cfg.OrderHint(),
			ProcessorConfig: runtime.RawExtension{Raw: configJSON},
		},
	}, nil
}

// SyncUrlTemplatizationProcessor creates, patches, or deletes the shared URL-templatization Processor from Actions.
//
// When mode is URLTemplatizationSyncCreateIfMissing: if the Processor already exists, returns without patching.
//
// When mode is URLTemplatizationSyncApplyFull: always builds and applies (or deletes)
// so the CR matches the desired state for URL templatization.
func SyncUrlTemplatizationProcessor(ctx context.Context, c client.Client, mode URLTemplatizationSyncMode) error {
	logger := commonlogger.FromContext(ctx).WithName("url-templatization")
	ns := env.GetCurrentNamespace()
	need, err := hasAnyUrlTemplatizationAction(ctx, c, ns)
	if err != nil {
		return err
	}
	if !need {
		proc := &odigosv1.Processor{ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      consts.URLTemplatizationProcessorName,
		}}
		err = c.Delete(ctx, proc)
		if err != nil {
			return client.IgnoreNotFound(err)
		}
		logger.Debug("shared url templatization processor removed",
			"reason", "no URL templatization action available, removed any stale entry")
		return nil
	}

	if mode == URLTemplatizationSyncCreateIfMissing {
		existing := &odigosv1.Processor{}
		key := client.ObjectKey{
			Namespace: ns,
			Name:      consts.URLTemplatizationProcessorName,
		}
		if err := c.Get(ctx, key, existing); err == nil {
			return nil
		}
	}

	nodeCG := &odigosv1.CollectorsGroup{}
	err = c.Get(ctx, client.ObjectKey{Namespace: ns, Name: k8sconsts.OdigosNodeCollectorCollectorGroupName}, nodeCG)
	spanMetricsEnabled := false
	if err == nil && nodeCG.Spec.Metrics != nil && nodeCG.Spec.Metrics.SpanMetrics != nil {
		spanMetricsEnabled = true
	}
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("get node CollectorsGroup: %w", err)
	}
	proc, err := buildUrlTemplatizationProcessor(ns, spanMetricsEnabled)
	if err != nil {
		return err
	}

	err = c.Patch(ctx, proc, client.Apply, client.FieldOwner("action-controller"), client.ForceOwnership)
	if err != nil {
		return err
	}
	return nil
}
