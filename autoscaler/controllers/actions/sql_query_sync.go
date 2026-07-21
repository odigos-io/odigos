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

// SQLQuerySyncMode selects how SyncSQLQueryProcessor reconciles the shared Processor CR.
type SQLQuerySyncMode int

const (
	// SQLQuerySyncCreateIfMissing: use from ActionReconcile only.
	// Ensures the shared Processor CR exists when missing, but if it already exists returns without patching.
	SQLQuerySyncCreateIfMissing SQLQuerySyncMode = 0
	// SQLQuerySyncApplyFull: use when the shared Processor must match
	// current Actions + node CollectorsGroup state.
	// Always builds and applies (or deletes when no SQL-query actions).
	SQLQuerySyncApplyFull SQLQuerySyncMode = 1
)

func hasAnySQLQueryAction(ctx context.Context, c client.Client, namespace string) (bool, error) {
	var list odigosv1.ActionList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return false, err
	}
	for i := range list.Items {
		a := &list.Items[i]
		if a.Spec.Disabled {
			continue
		}
		if a.Spec.DbQueryTemplatization != nil || a.Spec.InferDbAttributes != nil {
			return true, nil
		}
	}
	return false, nil
}

func buildSQLQueryProcessor(namespace string, spanMetricsEnabled bool) (*odigosv1.Processor, error) {
	cfg := actions.DbQueryTemplatizationConfig{}
	configJSON, err := json.Marshal(map[string]interface{}{
		"odigos_config_extension": k8sconsts.OdigosConfigK8sExtensionType,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal sql query processor config: %w", err)
	}
	// Span metrics are built in the node collector pipeline and label series using span name and
	// related attributes. Without processing SQL queries first, metrics explode in cardinality.
	// When span metrics are on, the shared processor must run on the node collector; otherwise
	// gateway-only is enough for traces.
	collectorRoles := cfg.SharedProcessorCollectorRoles(spanMetricsEnabled)
	roles := make([]odigosv1.CollectorsGroupRole, 0, len(collectorRoles))
	for _, r := range collectorRoles {
		roles = append(roles, odigosv1.CollectorsGroupRole(r))
	}
	return &odigosv1.Processor{
		TypeMeta: metav1.TypeMeta{APIVersion: "odigos.io/v1alpha1", Kind: "Processor"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.SQLQueryProcessorName,
			Namespace: namespace,
		},
		Spec: odigosv1.ProcessorSpec{
			Type:            cfg.ProcessorType(),
			ProcessorName:   "SQL Query",
			Disabled:        false,
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles:  roles,
			OrderHint:       cfg.OrderHint(),
			ProcessorConfig: runtime.RawExtension{Raw: configJSON},
		},
	}, nil
}

// SyncSQLQueryProcessor creates, patches, or deletes the shared SQL-query Processor from Actions.
//
// When mode is SQLQuerySyncCreateIfMissing: if the Processor already exists, returns without patching.
//
// When mode is SQLQuerySyncApplyFull: always builds and applies (or deletes)
// so the CR matches the desired state for SQL query processing.
func SyncSQLQueryProcessor(ctx context.Context, c client.Client, mode SQLQuerySyncMode) error {
	logger := commonlogger.FromContext(ctx).WithName("sql-query")
	ns := env.GetCurrentNamespace()
	need, err := hasAnySQLQueryAction(ctx, c, ns)
	if err != nil {
		return err
	}
	if !need {
		proc := &odigosv1.Processor{ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      consts.SQLQueryProcessorName,
		}}
		err = c.Delete(ctx, proc)
		if err != nil {
			return client.IgnoreNotFound(err)
		}
		logger.Debug("shared sql query processor removed",
			"reason", "no DbQueryTemplatization or InferDbAttributes action available, removed any stale entry")
		return nil
	}

	if mode == SQLQuerySyncCreateIfMissing {
		existing := &odigosv1.Processor{}
		key := client.ObjectKey{
			Namespace: ns,
			Name:      consts.SQLQueryProcessorName,
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
	proc, err := buildSQLQueryProcessor(ns, spanMetricsEnabled)
	if err != nil {
		return err
	}

	err = c.Patch(ctx, proc, client.Apply, client.FieldOwner("action-controller"), client.ForceOwnership)
	if err != nil {
		return err
	}
	return nil
}
