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

func piiMaskingConfigEmpty(cfg actionsapi.PiiMaskingConfig) bool {
	return len(cfg.PiiCategories) == 0 && len(cfg.CustomFormatMaskings) == 0 && len(cfg.CustomRegexMaskings) == 0
}

func collectPiiMaskingConfig(ctx context.Context, c client.Client, namespace string) (actionsapi.PiiMaskingConfig, error) {
	var list odigosv1.ActionList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return actionsapi.PiiMaskingConfig{}, err
	}

	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].Name < list.Items[j].Name
	})

	seenCategories := make(map[actionsapi.PiiCategory]struct{})
	seenFormats := make(map[string]struct{})
	seenRegexes := make(map[string]struct{})
	cfg := actionsapi.PiiMaskingConfig{}

	for i := range list.Items {
		a := &list.Items[i]
		if a.Spec.PiiMasking == nil || a.Spec.Disabled {
			continue
		}
		if !piiMaskingActionSignalsSupported(a.Spec.Signals) {
			continue
		}

		for _, category := range a.Spec.PiiMasking.PiiCategories {
			if _, ok := seenCategories[category]; ok {
				continue
			}
			seenCategories[category] = struct{}{}
			cfg.PiiCategories = append(cfg.PiiCategories, category)
		}

		for _, masking := range a.Spec.PiiMasking.CustomFormatMaskings {
			if masking.LookupKey == "" || masking.DataFormat == "" {
				continue
			}
			key := masking.LookupKey + "\x00" + string(masking.DataFormat)
			if _, ok := seenFormats[key]; ok {
				continue
			}
			seenFormats[key] = struct{}{}
			cfg.CustomFormatMaskings = append(cfg.CustomFormatMaskings, masking)
		}

		for _, masking := range a.Spec.PiiMasking.CustomRegexMaskings {
			if masking.Regex == "" {
				continue
			}
			if _, ok := seenRegexes[masking.Regex]; ok {
				continue
			}
			seenRegexes[masking.Regex] = struct{}{}
			cfg.CustomRegexMaskings = append(cfg.CustomRegexMaskings, masking)
		}
	}

	sort.Slice(cfg.PiiCategories, func(i, j int) bool {
		return cfg.PiiCategories[i] < cfg.PiiCategories[j]
	})

	return cfg, nil
}

// marshalPiiMaskingProcessorConfig renders the Action API config into the snake_case
// keys expected by the odigospiimasking processor mapstructure tags.
func marshalPiiMaskingProcessorConfig(cfg actionsapi.PiiMaskingConfig) ([]byte, error) {
	out := map[string]any{}
	if len(cfg.PiiCategories) > 0 {
		out["pii_categories"] = cfg.PiiCategories
	}
	if len(cfg.CustomFormatMaskings) > 0 {
		formats := make([]map[string]string, 0, len(cfg.CustomFormatMaskings))
		for _, masking := range cfg.CustomFormatMaskings {
			formats = append(formats, map[string]string{
				"lookup_key":  masking.LookupKey,
				"data_format": string(masking.DataFormat),
			})
		}
		out["custom_format_maskings"] = formats
	}
	if len(cfg.CustomRegexMaskings) > 0 {
		regexes := make([]map[string]string, 0, len(cfg.CustomRegexMaskings))
		for _, masking := range cfg.CustomRegexMaskings {
			regexes = append(regexes, map[string]string{
				"regex": masking.Regex,
			})
		}
		out["custom_regex_maskings"] = regexes
	}
	return json.Marshal(out)
}

func buildPiiMaskingProcessor(namespace string, cfg actionsapi.PiiMaskingConfig) (*odigosv1.Processor, error) {
	actionCfg := actions.PiiMaskingConfig{}
	configJSON, err := marshalPiiMaskingProcessorConfig(cfg)
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
// Categories and custom masking rules from all non-disabled PiiMasking actions are unioned into a
// single processor. When no enabled actions remain, the shared Processor is deleted.
// Legacy per-action Processor CRs (named after the Action) are also removed.
func SyncPiiMaskingProcessor(ctx context.Context, c client.Client) error {
	logger := commonlogger.FromContext(ctx).WithName("pii-masking")
	ns := env.GetCurrentNamespace()

	if err := deleteLegacyPiiMaskingProcessors(ctx, c, ns); err != nil {
		return err
	}

	cfg, err := collectPiiMaskingConfig(ctx, c, ns)
	if err != nil {
		return err
	}
	if piiMaskingConfigEmpty(cfg) {
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

	proc, err := buildPiiMaskingProcessor(ns, cfg)
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
