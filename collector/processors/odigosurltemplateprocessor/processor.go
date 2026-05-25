package odigosurltemplateprocessor

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	deprecatedsemconv "go.opentelemetry.io/collector/semconv/v1.18.0"
	semconv "go.opentelemetry.io/collector/semconv/v1.27.0"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/collector"
)

// Ensure urlTemplateProcessor implements the callback interface used by the extension.
var _ collector.WorkloadConfigCacheCallback = (*urlTemplateProcessor)(nil)

// workloadUrlTemplatizationConfig holds the result of parsing URL templatization rules for one workload/container.
// Stored in processorURLTemplateParsedRulesCache so we parse once per entry, not per batch.
type workloadUrlTemplatizationConfig struct {

	// the rules to apply for templatization for this workload.
	parsedRules map[int][]TemplatizationRule // nil means heuristic-only (no explicit rules)

	// if true, skip default templatization for non-successful responses.
	// this is used to avoid high-cardinality of templated routes
	// when an endpoint from this service returns with 404 status code.
	// internet exposed services are commonly being "tested" by malicious actors
	// with irrelevant or garbage requests that can contaminate the url-templatization process
	// leading to high-cardinality of templated routes.
	avoidDefaultTemplatizationOnError bool
}

type urlTemplateProcessor struct {
	logger              *zap.Logger
	cfg                 *Config
	templatizationRules map[int][]TemplatizationRule // group templatization rules by segments length
	customIds           []internalCustomIdConfig

	excludeMatcher *PropertiesMatcher
	includeMatcher *PropertiesMatcher

	// provider is set in Start() when odigos_config_extension is present (the default in Odigos-managed configs).
	// Per-workload rules come from the extension cache; include/exclude matchers apply only on the legacy static path.
	provider collector.OdigosConfigExtension

	// processorURLTemplateParsedRulesCache caches parsed rules per workload key; updated via extension callback.
	parsedRulesCache *processorURLTemplateParsedRulesCache
}

func newUrlTemplateProcessor(set processor.Settings, config *Config) (*urlTemplateProcessor, error) {

	excludeMatcher := NewPropertiesMatcher(config.Exclude)
	includeMatcher := NewPropertiesMatcher(config.Include)

	parsedRules := map[int][]TemplatizationRule{}
	for _, rule := range config.TemplatizationRules {
		parsedRule, err := parseUserInputRuleString(rule)
		if err != nil {
			return nil, err
		}
		parsedRuleNumSegments := len(parsedRule)
		if _, ok := parsedRules[parsedRuleNumSegments]; !ok {
			parsedRules[parsedRuleNumSegments] = []TemplatizationRule{}
		}
		parsedRules[parsedRuleNumSegments] = append(parsedRules[parsedRuleNumSegments], parsedRule)
	}

	customIdsRegexp := make([]internalCustomIdConfig, 0, len(config.CustomIds))
	for _, ci := range config.CustomIds {
		regexpPattern, err := regexp.Compile(ci.Regexp)
		if err != nil {
			return nil, fmt.Errorf("invalid custom id regex: %w", err)
		}
		templateName := "id"
		if ci.TemplateName != "" {
			// if the template name is empty, we default to "id"
			templateName = ci.TemplateName
		}
		customIdsRegexp = append(customIdsRegexp, internalCustomIdConfig{
			Regexp: *regexpPattern,
			Name:   templateName,
		})
	}

	return &urlTemplateProcessor{
		logger:              set.Logger,
		cfg:                 config,
		templatizationRules: parsedRules,
		customIds:           customIdsRegexp,
		excludeMatcher:      excludeMatcher,
		includeMatcher:      includeMatcher,
		parsedRulesCache:    newProcessorURLTemplateParsedRulesCache(),
	}, nil
}

// Start resolves odigos_config_extension (default in Odigos) and registers for workload config updates.
func (p *urlTemplateProcessor) Start(ctx context.Context, host component.Host) error {
	if p.cfg.OdigosConfigExtension == nil {
		p.logger.Warn("odigos_config_extension unset, ensure processor contains the templatization rules")
		return nil
	}
	extID := p.cfg.OdigosConfigExtension
	extensions := host.GetExtensions()
	if ext, ok := extensions[*extID]; ok {
		return p.registerOdigosConfigExtension(ctx, ext, extID.String())
	}
	return fmt.Errorf("odigos config extension %q not found or no instance implements OdigosConfigExtension", extID.String())
}

func (p *urlTemplateProcessor) registerOdigosConfigExtension(ctx context.Context, ext component.Component, extensionID string) error {
	odigosExt, ok := ext.(collector.OdigosConfigExtension)
	if !ok {
		return fmt.Errorf("extension %q is not an OdigosConfigExtension (got %T)", extensionID, ext)
	}
	p.provider = odigosExt
	odigosExt.RegisterWorkloadConfigCacheCallback(p)
	if !p.provider.WaitForCacheSync(ctx) {
		p.logger.Warn("odigos config extension cache sync did not complete; some spans may be missed on startup")
	}
	return nil
}

// Shutdown unregisters from the extension and clears local caches.
func (p *urlTemplateProcessor) Shutdown(context.Context) error {
	if p.provider != nil {
		p.provider.UnregisterWorkloadConfigCacheCallback(p)
		p.provider = nil
	}
	p.parsedRulesCache.clear()
	return nil
}

// OnSet implements collector.WorkloadConfigCacheCallback; called when the extension cache adds/updates an entry.
// Empty or nil rules: store entry with parsedRules=nil so the workload gets default heuristic templatization (same as when extension is disabled).
func (p *urlTemplateProcessor) OnSet(key string, cfg *commonapi.ContainerCollectorConfig) {
	avoidDefaultTemplatizationOnError := cfg.UrlTemplatization.AvoidDefaultTemplatizationOnError
	if cfg.UrlTemplatization == nil || len(cfg.UrlTemplatization.TemplatizationRules) == 0 {
		p.parsedRulesCache.set(key, workloadUrlTemplatizationConfig{
			parsedRules:                       nil,
			avoidDefaultTemplatizationOnError: avoidDefaultTemplatizationOnError,
		})
		p.logger.Debug("workload config cache OnSet: no rules, use default heuristic", zap.String("key", key))
		return
	}
	parsedRules := p.parseRuleStrings(cfg.UrlTemplatization.TemplatizationRules)
	p.parsedRulesCache.set(key, workloadUrlTemplatizationConfig{
		parsedRules:                       parsedRules,
		avoidDefaultTemplatizationOnError: avoidDefaultTemplatizationOnError,
	})
	p.logger.Debug("workload config cache OnSet", zap.String("key", key))
}

// OnDeleteKey implements collector.WorkloadConfigCacheCallback; called when the extension cache removes an entry.
func (p *urlTemplateProcessor) OnDeleteKey(key string) {
	p.parsedRulesCache.delete(key)
	p.logger.Debug("workload config cache OnDeleteKey", zap.String("key", key))
}

// parseRuleStrings parses a slice of rule strings into a map of segment-count → rules.
// Each string is parsed via parseUserInputRuleString; invalid rules are skipped with a warning.
func (p *urlTemplateProcessor) parseRuleStrings(ruleStrings []string) map[int][]TemplatizationRule {
	parsed := map[int][]TemplatizationRule{}
	for _, rule := range ruleStrings {
		parsedRule, err := parseUserInputRuleString(rule)
		if err != nil {
			p.logger.Warn("invalid templatization rule; skipping", zap.String("rule", rule), zap.Error(err))
			continue
		}
		n := len(parsedRule)
		parsed[n] = append(parsed[n], parsedRule)
	}
	return parsed
}

func (p *urlTemplateProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	resourceSpanCount := td.ResourceSpans().Len()
	p.logger.Debug("processTraces started", zap.Int("resource_spans", resourceSpanCount))
	for i := 0; i < resourceSpanCount; i++ {
		resourceSpans := td.ResourceSpans().At(i)

		if p.provider != nil {
			key, err := p.provider.GetWorkloadCacheKey(resourceSpans.Resource())
			if err != nil {
				p.logger.Debug("processTraces skip resource: GetWorkloadCacheKey failed", zap.Error(err))
				continue
			}
			spanUrlTemplatizationConfig, ok := p.parsedRulesCache.get(key)
			if !ok {
				// Rely entirely on the extension callback to populate the cache; skip this resource until we have an entry.
				continue
			}
			// entry.parsedRules may be nil: extension sent no rules → use default heuristic only (defaultTemplatizeURLPath).
			for j := 0; j < resourceSpans.ScopeSpans().Len(); j++ {
				scopeSpans := resourceSpans.ScopeSpans().At(j)
				for k := 0; k < scopeSpans.Spans().Len(); k++ {
					span := scopeSpans.Spans().At(k)
					p.processSpanWithRules(span, spanUrlTemplatizationConfig)
				}
			}
		} else {
			if p.excludeMatcher != nil && p.excludeMatcher.Match(resourceSpans.Resource()) {
				continue
			}
			if p.includeMatcher != nil && !p.includeMatcher.Match(resourceSpans.Resource()) {
				continue
			}
			for j := 0; j < resourceSpans.ScopeSpans().Len(); j++ {
				scopeSpans := resourceSpans.ScopeSpans().At(j)
				for k := 0; k < scopeSpans.Spans().Len(); k++ {
					span := scopeSpans.Spans().At(k)
					p.processSpanWithStaticRules(span)
				}
			}
		}
	}
	return td, nil
}

func getHttpMethod(attr pcommon.Map) (string, bool) {
	// prefer to use the new "http.request.method" attribute
	if method, found := attr.Get(semconv.AttributeHTTPRequestMethod); found {
		return method.AsString(), true
	}
	// fallback to the old "http.method" attribute which might still be used
	// by some instrumentations.
	// TODO: remove this fallback in the future when all instrumentations are aligned with
	// update semantic conventions and no longer report "http.method"
	if method, found := attr.Get(deprecatedsemconv.AttributeHTTPMethod); found {
		return method.AsString(), true
	}
	return "", false
}

func getUrlPath(attr pcommon.Map) (string, bool) {

	// prefer the updated semantic convention "url.path" if available
	if urlPath, found := attr.Get(semconv.AttributeURLPath); found {
		return urlPath.AsString(), true
	}

	// fallback to the old "http.target" attribute which might still be used
	// by some instrumentations.
	// TODO: remove this fallback in the future when all instrumentations are aligned with
	// update semantic conventions and no longer report "http.target"
	if httpTarget, found := attr.Get(deprecatedsemconv.AttributeHTTPTarget); found {
		// the "http.target" attribute might contain a query string, so we need to
		// split it and only use the path part.
		// for example: "/user?id=123" => "/user"
		path := strings.SplitN(httpTarget.AsString(), "?", 2)[0]
		return path, true
	}
	return "", false
}

func getFullUrl(attr pcommon.Map) (string, bool) {
	// prefer the updated semantic convention "url.full" if available
	if fullUrl, found := attr.Get(semconv.AttributeURLFull); found {
		return fullUrl.AsString(), true
	}
	// fallback to the old "http.url" attribute which might still be used
	// by some instrumentations.
	// TODO: remove this fallback in the future when all instrumentations are aligned with
	// update semantic conventions and no longer report "http.url"
	if httpUrl, found := attr.Get(deprecatedsemconv.AttributeHTTPURL); found {
		return httpUrl.AsString(), true
	}
	return "", false
}

// resolves a url path from path attribute or full url attribute.
func resolveUrlPath(attr pcommon.Map) (string, bool) {
	urlPath, urlPathFound := getUrlPath(attr)
	if urlPathFound {
		return urlPath, true
	}
	fullUrl, fullUrlFound := getFullUrl(attr)
	if fullUrlFound {
		parsed, err := url.Parse(fullUrl)
		if err != nil {
			return "", false
		}
		return parsed.Path, true
	}
	return "", false
}

func getHttpResponseStatusCode(attr pcommon.Map) (int, bool) {
	if statusCode, found := attr.Get(semconv.AttributeHTTPResponseStatusCode); found {
		if statusCode.Type() != pcommon.ValueTypeInt {
			return 0, false
		}
		return int(statusCode.Int()), true
	}
	// fallback to the old "http.status_code" attribute which might still be used
	if statusCode, found := attr.Get(deprecatedsemconv.AttributeHTTPStatusCode); found {
		if statusCode.Type() != pcommon.ValueTypeInt {
			return 0, false
		}
		return int(statusCode.Int()), true
	}
	return 0, false
}

func splitPathToSegments(path string) ([]string, bool) {
	hasLeadingSlash := strings.HasPrefix(path, "/")
	if !hasLeadingSlash {
		path = "/" + path
	}

	inputPathSegments := strings.Split(path, "/")
	inputPathSegments = inputPathSegments[1:]
	return inputPathSegments, hasLeadingSlash
}

// calculateTemplatedUrlFromAttrWithRules calculates a templated URL using the given rules.
func (p *urlTemplateProcessor) calculateTemplatedUrlFromAttrWithRules(attr pcommon.Map, config workloadUrlTemplatizationConfig) (string, bool) {
	urlPath, urlPathFound := resolveUrlPath(attr)
	if !urlPathFound {
		return "", false
	}

	// M7: normalize paths that are all slashes (e.g. "//", "///") to "/"
	if strings.Trim(urlPath, "/") == "" {
		p.logger.Debug("applyTemplatizationOnPath: all-slashes normalized to /", zap.String("path", urlPath))
		return "/", true
	}

	inputPathSegments, hadLeadingSlash := splitPathToSegments(urlPath)
	if len(inputPathSegments) == 1 && inputPathSegments[0] == "" {
		// if the path is empty, we can't generate a templated url
		return "/", true // always set a leading slash even if missing
	}

	// attempt the rules if we have any
	if config.parsedRules != nil {
		templatedUrl, matched := applyCustomRulesForTemplatization(inputPathSegments, config.parsedRules, hadLeadingSlash)
		if matched {
			return templatedUrl, true
		}
	}

	// check for malicious bots routes so not to templatize them.
	// do it after the custom rules to allow legitimate routes to be templatized even for services with http errors.
	if config.avoidDefaultTemplatizationOnError {
		statusCode, found := getHttpResponseStatusCode(attr)
		if found {
			// currently evaluating only 404 status code.
			// in the future we might want to extend it to cover more cases or make it configurable.
			if statusCode == 404 {
				p.logger.Debug("applyTemplatizationOnPath: 404 status code, skip templatization", zap.String("path", urlPath))
				return "", false
			}
		}
	}

	// default templatization
	templatedPath, isTemplated := defaultTemplatizeURLPath(inputPathSegments, p.customIds)
	if isTemplated {
		if hadLeadingSlash {
			// if the path has a leading slash, we need to add it back
			templatedPath = "/" + templatedPath
		}
		return templatedPath, true
	}

	p.logger.Debug("applyTemplatizationOnPath: no match, path unchanged", zap.String("path", urlPath))
	return urlPath, true
}

func updateHttpSpanName(span ptrace.Span, httpMethod string, templatedUrl string) {
	currentName := span.Name()
	if currentName != httpMethod {
		// be conservative and only update the name for the use case "GET" => "GET /user/{id}"
		// if the span name is set to something else, keep it and don't override it.
		// we might want to revisit this in the future based on real world feedback.
		return
	}

	// if the templated url is not available, we keep the span name as is.
	if templatedUrl == "" {
		return
	}

	// generate span name based on semantic conventions:
	// HTTP span names SHOULD be {method} {target} if there is a (low-cardinality) target available.
	// the "target" in our case is the templated url (which is either http.route or url.template attributes).
	newSpanName := fmt.Sprintf("%s %s", httpMethod, templatedUrl)
	span.SetName(newSpanName)
}

func (p *urlTemplateProcessor) enhanceSpanWithRules(span ptrace.Span, httpMethod string, targetAttribute string, config workloadUrlTemplatizationConfig) {
	attr := span.Attributes()

	// edge case: target attribute (http.route) exists but is empty (e.g. no path)
	// in this case, we align and normalize the value to "/" to denote that.
	if val, found := attr.Get(targetAttribute); found {
		if val.Type() != pcommon.ValueTypeStr {
			// should not happen.
			return
		}
		if val.Str() == "" {
			updateHttpSpanName(span, httpMethod, "/")
		}
		// avoid overriding the attribute if it is already set
		return
	}

	templatedUrl, found := p.calculateTemplatedUrlFromAttrWithRules(attr, config)
	if !found {
		p.logger.Debug("enhanceSpanWithRules: no url/path in attributes or publically accessible service with http error, skip", zap.String("span_name", span.Name()))
		return
	}

	// set the templated url in the target attribute and update the span name if needed
	attr.PutStr(targetAttribute, templatedUrl)
	updateHttpSpanName(span, httpMethod, templatedUrl)
}

// processSpanWithRules enhances an HTTP span with templated URL using the given rules.
// processSpanWithStaticRules uses static config rules; extension path uses per-workload rules.
func (p *urlTemplateProcessor) processSpanWithRules(span ptrace.Span, config workloadUrlTemplatizationConfig) {
	attr := span.Attributes()

	httpMethod, found := getHttpMethod(attr)
	if !found {
		// we only enhance http spans, so if there is no http.method attribute, we can skip it
		return
	}

	switch span.Kind() {

	case ptrace.SpanKindClient:
		// client spans write the url templated value in "url.template" attribute.
		p.enhanceSpanWithRules(span, httpMethod, semconv.AttributeURLTemplate, config)
	case ptrace.SpanKindServer:
		// server spans write the url templated value in "http.route" attribute.
		p.enhanceSpanWithRules(span, httpMethod, semconv.AttributeHTTPRoute, config)
	default:
		// http spans are either client or server
		// all other spans are ignored and never enhanced
		return
	}
}

func (p *urlTemplateProcessor) processSpanWithStaticRules(span ptrace.Span) {
	// for static mode, we do not set publiclyAccessible flag.
	// it is only used with extension mode.
	// this code will be removed once we fully migrate to extension mode.
	p.processSpanWithRules(span, workloadUrlTemplatizationConfig{parsedRules: p.templatizationRules, avoidDefaultTemplatizationOnError: false})
}
