package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func collectPiiMaskingCategories(ctx context.Context, c client.Client, namespace string) ([]actionsapi.PiiCategory, error) {
	var list odigosv1.ActionList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	seen := make(map[actionsapi.PiiCategory]struct{})
	for i := range list.Items {
		a := &list.Items[i]
		if a.Spec.PiiMasking == nil || a.Spec.Disabled {
			continue
		}
		if !piiMaskingActionSignalsSupported(a.Spec.Signals) {
			continue
		}
		for _, category := range a.Spec.PiiMasking.PiiCategories {
			seen[category] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil, nil
	}

	categories := make([]actionsapi.PiiCategory, 0, len(seen))
	for category := range seen {
		categories = append(categories, category)
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i] < categories[j]
	})
	return categories, nil
}

func buildPiiMaskingProcessor(namespace string, categories []actionsapi.PiiCategory) (*odigosv1.Processor, error) {
	cfg := actions.PiiMaskingConfig{}
	configJSON, err := json.Marshal(piiMaskingProcessorConfig{
		PiiCategories: categories,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal pii masking processor config: %w", err)
	}

	collectorRoles := make([]odigosv1.CollectorsGroupRole, 0, len(cfg.CollectorRoles()))
	for _, r := range cfg.CollectorRoles() {
		collectorRoles = append(collectorRoles, odigosv1.CollectorsGroupRole(r))
	}

	return &odigosv1.Processor{
		TypeMeta: metav1.TypeMeta{APIVersion: "odigos.io/v1alpha1", Kind: "Processor"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.PiiMaskingProcessorName,
			Namespace: namespace,
		},
		Spec: odigosv1.ProcessorSpec{
			Type:            cfg.ProcessorType(),
			ProcessorName:   "PII Masking",
			Disabled:        false,
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles:  collectorRoles,
			OrderHint:       cfg.OrderHint(),
			ProcessorConfig: runtime.RawExtension{Raw: configJSON},
		},
	}, nil
}

// SyncPiiMaskingProcessor creates, patches, or deletes the shared PII-masking Processor from Actions.
// Categories from all non-disabled PiiMasking actions are unioned into a single processor.
// When no enabled actions remain, the shared Processor is deleted.
// Legacy per-action Processor CRs (named after the Action) are also removed.
func SyncPiiMaskingProcessor(ctx context.Context, c client.Client) error {
	logger := commonlogger.FromContext(ctx).WithName("pii-masking")
	ns := env.GetCurrentNamespace()

	if err := deleteLegacyPiiMaskingProcessors(ctx, c, ns); err != nil {
		return err
	}

	categories, err := collectPiiMaskingCategories(ctx, c, ns)
	if err != nil {
		return err
	}
	if len(categories) == 0 {
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

	proc, err := buildPiiMaskingProcessor(ns, categories)
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
