package odigospiimaskingprocessor

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/collector"
)

// Ensure piiMaskingProcessor implements the callback interface used by the extension.
var _ collector.WorkloadConfigCacheCallback = (*piiMaskingProcessor)(nil)

type compiledPiiMaskingConfig struct {
	categories    []actions.PiiCategory
	customMaskers []*regexp.Regexp
}

type piiMaskingProcessor struct {
	logger *zap.Logger
	cfg    *Config

	// provider is set in Start() from odigos_config_extension.
	provider collector.OdigosConfigExtension

	// maskersCache caches compiled rules per workload key; updated via extension callback.
	maskersCache *processorPiiMaskingCache
}

func newPiiMaskingProcessor(set processor.Settings, cfg *Config) *piiMaskingProcessor {
	return &piiMaskingProcessor{
		logger:       set.Logger,
		cfg:          cfg,
		maskersCache: newProcessorPiiMaskingCache(),
	}
}

func compilePiiMaskingConfig(cfg *actions.PiiMaskingConfig) (compiledPiiMaskingConfig, error) {
	compiled := compiledPiiMaskingConfig{
		categories: make([]actions.PiiCategory, 0, len(cfg.PiiCategories)),
	}
	var errs error

	for i, category := range cfg.PiiCategories {
		if _, ok := categoryMasks[category]; !ok {
			errs = errors.Join(errs, fmt.Errorf("piiCategories[%d]: unsupported category %q", i, category))
			continue
		}
		compiled.categories = append(compiled.categories, category)
	}

	customMaskers, err := compileCustomMaskers(cfg)
	compiled.customMaskers = customMaskers
	return compiled, errors.Join(errs, err)
}

func compileCustomMaskers(cfg *actions.PiiMaskingConfig) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(cfg.CustomFormatMaskings)+len(cfg.CustomRegexMaskings))
	var errs error

	for i, masking := range cfg.CustomFormatMaskings {
		if masking.LookupKey == "" {
			errs = errors.Join(errs, fmt.Errorf("customFormatMaskings[%d]: lookupKey is required", i))
			continue
		}
		re, err := buildFormatMaskingRegex(masking.LookupKey, masking.DataFormat)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("customFormatMaskings[%d]: %w", i, err))
			continue
		}
		out = append(out, re)
	}

	for i, masking := range cfg.CustomRegexMaskings {
		if masking.Regex == "" {
			errs = errors.Join(errs, fmt.Errorf("customRegexMaskings[%d]: regex is required", i))
			continue
		}
		re, err := regexp.Compile(masking.Regex)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("customRegexMaskings[%d]: invalid regex: %w", i, err))
			continue
		}
		if re.NumSubexp() < 1 {
			errs = errors.Join(errs, fmt.Errorf("customRegexMaskings[%d]: regex must contain at least one capture group", i))
			continue
		}
		out = append(out, re)
	}

	return out, errs
}

// Start resolves odigos_config_extension for per-source config lookups.
func (p *piiMaskingProcessor) Start(ctx context.Context, host component.Host) error {
	if p.cfg.OdigosConfigExtension == nil {
		return fmt.Errorf("odigos_config_extension is required")
	}
	extID := p.cfg.OdigosConfigExtension
	ext, ok := host.GetExtensions()[*extID]
	if !ok {
		return fmt.Errorf("odigos config extension %q not found", extID.String())
	}
	odigosExt, ok := ext.(collector.OdigosConfigExtension)
	if !ok {
		return fmt.Errorf("extension %q is not an OdigosConfigExtension (got %T)", extID.String(), ext)
	}
	p.provider = odigosExt
	odigosExt.RegisterWorkloadConfigCacheCallback(p)
	if !p.provider.WaitForCacheSync(ctx) {
		p.logger.Warn("odigos config extension cache sync did not complete; some spans may be missed on startup")
	}
	return nil
}

// Shutdown unregisters from the extension and clears local caches.
func (p *piiMaskingProcessor) Shutdown(context.Context) error {
	if p.provider != nil {
		p.provider.UnregisterWorkloadConfigCacheCallback(p)
		p.provider = nil
	}
	p.maskersCache.clear()
	return nil
}

// OnSet implements collector.WorkloadConfigCacheCallback.
func (p *piiMaskingProcessor) OnSet(key string, cfg *commonapi.ContainerCollectorConfig) {
	if cfg.PiiMasking == nil {
		p.maskersCache.delete(key)
		return
	}

	compiled, err := compilePiiMaskingConfig(cfg.PiiMasking)
	if err != nil {
		p.logger.Warn("invalid pii masking rules; skipping malformed rules", zap.String("key", key), zap.Error(err))
	}
	if len(compiled.categories) == 0 && len(compiled.customMaskers) == 0 {
		if err == nil {
			p.maskersCache.delete(key)
		}
		return
	}
	p.maskersCache.set(key, compiled)
	p.logger.Debug("workload config cache OnSet", zap.String("key", key))
}

// OnDeleteKey implements collector.WorkloadConfigCacheCallback.
func (p *piiMaskingProcessor) OnDeleteKey(key string) {
	p.maskersCache.delete(key)
	p.logger.Debug("workload config cache OnDeleteKey", zap.String("key", key))
}

func (p *piiMaskingProcessor) processTraces(_ context.Context, traces ptrace.Traces) (ptrace.Traces, error) {
	if p.provider == nil {
		return traces, nil
	}

	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)

		key, err := p.provider.GetWorkloadCacheKey(rs.Resource())
		if err != nil {
			continue
		}
		maskCfg, ok := p.maskersCache.get(key)
		if !ok {
			continue
		}

		scopeSpans := rs.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				p.processSpan(spans.At(k), maskCfg)
			}
		}
	}
	return traces, nil
}

func (p *piiMaskingProcessor) processSpan(span ptrace.Span, cfg compiledPiiMaskingConfig) {
	span.Attributes().Range(func(_ string, value pcommon.Value) bool {
		p.processAttributeValue(value, cfg)
		return true
	})
}

func (p *piiMaskingProcessor) processAttributeValue(value pcommon.Value, cfg compiledPiiMaskingConfig) {
	switch value.Type() {
	case pcommon.ValueTypeStr:
		if masked, changed := maskPiiData(value.Str(), cfg); changed {
			value.SetStr(masked)
		}
	case pcommon.ValueTypeSlice:
		slice := value.Slice()
		for i := 0; i < slice.Len(); i++ {
			p.processAttributeValue(slice.At(i), cfg)
		}
	}
}

func maskPiiData(value string, cfg compiledPiiMaskingConfig) (string, bool) {
	result := value
	changed := false

	for _, category := range cfg.categories {
		masked, applied := maskCategory(category, result)
		if applied {
			result = masked
			changed = true
		}
	}

	for _, re := range cfg.customMaskers {
		masked, applied := maskCaptureGroups(re, result)
		if applied {
			result = masked
			changed = true
		}
	}

	return result, changed
}
