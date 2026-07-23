package actions

import (
	"context"
	"encoding/json"
	"fmt"

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

func hasAnyPiiMaskingAction(ctx context.Context, c client.Client, namespace string) (bool, error) {
	var list odigosv1.ActionList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return false, err
	}
	for i := range list.Items {
		a := &list.Items[i]
		if a.Spec.Disabled || a.Spec.PiiMasking == nil {
			continue
		}
		if !piiMaskingActionSignalsSupported(a.Spec.Signals) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func buildPiiMaskingProcessor(namespace string) (*odigosv1.Processor, error) {
	actionCfg := actions.PiiMaskingConfig{}
	configJSON, err := json.Marshal(map[string]interface{}{
		"odigos_config_extension": k8sconsts.OdigosConfigK8sExtensionType,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal pii masking processor config: %w", err)
	}

	collectorRoles := make([]odigosv1.CollectorsGroupRole, 0, len(actionCfg.CollectorRoles()))
	for _, r := range actionCfg.CollectorRoles() {
		collectorRoles = append(collectorRoles, odigosv1.CollectorsGroupRole(r))
	}

	return &odigosv1.Processor{
		TypeMeta: metav1.TypeMeta{APIVersion: "odigos.io/v1alpha1", Kind: "Processor"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.PiiMaskingProcessorName,
			Namespace: namespace,
		},
		Spec: odigosv1.ProcessorSpec{
			Type:            actionCfg.ProcessorType(),
			ProcessorName:   "PII Masking",
			Disabled:        false,
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles:  collectorRoles,
			OrderHint:       actionCfg.OrderHint(),
			ProcessorConfig: runtime.RawExtension{Raw: configJSON},
		},
	}, nil
}

// SyncPiiMaskingProcessor creates, patches, or deletes the shared PII-masking Processor from Actions.
// When any enabled PiiMasking action exists, the processor is configured with odigos_config_extension
// so per-source rules come from InstrumentationConfig. When no enabled actions remain, the shared
// Processor is deleted. Legacy per-action Processor CRs (named after the Action) are also removed.
func SyncPiiMaskingProcessor(ctx context.Context, c client.Client) error {
	logger := commonlogger.FromContext(ctx).WithName("pii-masking")
	ns := env.GetCurrentNamespace()

	if err := deleteLegacyPiiMaskingProcessors(ctx, c, ns); err != nil {
		return err
	}

	need, err := hasAnyPiiMaskingAction(ctx, c, ns)
	if err != nil {
		return err
	}
	if !need {
		proc := &odigosv1.Processor{ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      consts.PiiMaskingProcessorName,
		}}
		err = c.Delete(ctx, proc)
		if err != nil {
			return client.IgnoreNotFound(err)
		}
		logger.Debug("shared pii masking processor removed",
			"reason", "no PII masking action available, removed any stale entry")
		return nil
	}

	proc, err := buildPiiMaskingProcessor(ns)
	if err != nil {
		return err
	}

	return c.Patch(ctx, proc, client.Apply, client.FieldOwner("action-controller"), client.ForceOwnership)
}

// deleteLegacyPiiMaskingProcessors removes per-action Processor CRs created before PII masking
// used a single shared Processor. Collector keys look like odigospiimasking/<action-name>.
func deleteLegacyPiiMaskingProcessors(ctx context.Context, c client.Client, namespace string) error {
	var list odigosv1.ProcessorList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return err
	}
	for i := range list.Items {
		proc := &list.Items[i]
		if proc.Spec.Type != consts.OdigosPiiMaskingProcessorType {
			continue
		}
		if proc.Name == consts.PiiMaskingProcessorName {
			continue
		}
		if err := c.Delete(ctx, proc); err != nil {
			return client.IgnoreNotFound(err)
		}
	}
	return nil
}
