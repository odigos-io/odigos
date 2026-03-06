package odigosurltemplateprocessor

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	deprecatedsemconv "go.opentelemetry.io/collector/semconv/v1.18.0"
	semconv "go.opentelemetry.io/collector/semconv/v1.27.0"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
)

// parsedWorkloadEntry holds the result of parsing URL templatization rules for one workload/container.
// Stored in processorURLTemplateParsedRulesCache so we parse once per entry, not per batch.
type parsedWorkloadEntry struct {
	parsedRules map[int][]TemplatizationRule // nil means heuristic-only (no explicit rules)
}

type urlTemplateProcessor struct {
	logger              *zap.Logger
	templatizationRules map[int][]TemplatizationRule // group templatization rules by segments length
	customIds           []internalCustomIdConfig

	excludeMatcher *PropertiesMatcher
	includeMatcher *PropertiesMatcher

	// provider is optionally injected by the extensionStartWrapper at Start() time.
	// When set, per-workload rules are fetched from the extension cache and the
	// static include/exclude matchers are bypassed.
	provider workloadRulesProvider

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
		templatizationRules: parsedRules,
		customIds:           customIdsRegexp,
		excludeMatcher:      excludeMatcher,
		includeMatcher:      includeMatcher,
		parsedRulesCache:    newProcessorURLTemplateParsedRulesCache(),
	}, nil
}

// OnSet implements the extension's UrlTemplatizationCacheCallback; called when the extension cache adds/updates an entry.
func (p *urlTemplateProcessor) OnSet(key string, cfg *commonapi.ContainerCollectorConfig) {
	hasRules := cfg.UrlTemplatization != nil && len(cfg.UrlTemplatization.TemplatizationRules) > 0
	var parsedRules map[int][]TemplatizationRule
	if hasRules {
		parsedRules = p.parseRuleStrings(cfg.UrlTemplatization.TemplatizationRules)
	}
	p.parsedRulesCache.set(key, parsedWorkloadEntry{parsedRules: parsedRules})
	p.logger.Debug("url templatization cache OnSet", zap.String("key", key), zap.Bool("has_rules", hasRules))
}

// OnDeleteKey implements the extension's UrlTemplatizationCacheCallback; called when the extension cache removes an entry.
func (p *urlTemplateProcessor) OnDeleteKey(key string) {
	p.parsedRulesCache.delete(key)
	p.logger.Debug("url templatization cache OnDeleteKey", zap.String("key", key))
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
			attrs := resourceSpans.Resource().Attributes()
			key, err := p.provider.GetWorkloadCacheKey(attrs)
			if err != nil {
				p.logger.Debug("processTraces skip resource: GetWorkloadCacheKey failed", zap.Error(err))
				continue
			}
			entry, ok := p.parsedRulesCache.get(key)
			if !ok {
				// Cache miss: fetch from extension, parse once, store, then use.
				rules := p.provider.GetWorkloadUrlTemplatizationRules(attrs)
				var parsedRules map[int][]TemplatizationRule
				if len(rules) > 0 {
					parsedRules = p.parseRuleStrings(rules)
					entry = parsedWorkloadEntry{parsedRules: parsedRules}
					p.parsedRulesCache.set(key, entry)
				}
			}
			if entry.parsedRules == nil {
				continue
			}
			for j := 0; j < resourceSpans.ScopeSpans().Len(); j++ {
				scopeSpans := resourceSpans.ScopeSpans().At(j)
				for k := 0; k < scopeSpans.Spans().Len(); k++ {
					span := scopeSpans.Spans().At(k)
					p.processSpanWithRules(span, entry.parsedRules)
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
					p.processSpan(span)
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

// applyTemplatizationOnPath applies URL templatization to a path using the given parsed rules and custom IDs.
// M7: paths that consist only of slashes are normalized to "/" immediately.
func (p *urlTemplateProcessor) applyTemplatizationOnPathWithRules(path string, rules map[int][]TemplatizationRule, customIds []internalCustomIdConfig) string {
	// M7: normalize paths that are all slashes (e.g. "//", "///") to "/"
	if strings.Trim(path, "/") == "" {
		p.logger.Debug("applyTemplatizationOnPath: all-slashes normalized to /", zap.String("path", path))
		return "/"
	}

	hasLeadingSlash := strings.HasPrefix(path, "/")
	if !hasLeadingSlash {
		path = "/" + path
	}

	inputPathSegments := strings.Split(path, "/")
	inputPathSegments = inputPathSegments[1:]
	if len(inputPathSegments) == 1 && inputPathSegments[0] == "" {
		// if the path is empty, we can't generate a templated url
		return "/" // always set a leading slash even if missing
	}

	if rules != nil {
		ruleList, found := rules[len(inputPathSegments)]
		if found {
			for _, rule := range ruleList {
				if templatedUrl, matched := attemptTemplateWithRule(inputPathSegments, rule); matched {
					if hasLeadingSlash {
						templatedUrl = "/" + templatedUrl
					}
					return templatedUrl
				}
			}
		}
	}

	templatedPath, isTemplated := defaultTemplatizeURLPath(inputPathSegments, customIds)
	if isTemplated {
		if hasLeadingSlash {
			templatedPath = "/" + templatedPath
		}
		return templatedPath
	}
	p.logger.Debug("applyTemplatizationOnPath: no match, path unchanged", zap.String("path", path))
	return path
}

func (p *urlTemplateProcessor) applyTemplatizationOnPath(path string) string {
	return p.applyTemplatizationOnPathWithRules(path, p.templatizationRules, p.customIds)
}

func (p *urlTemplateProcessor) calculateTemplatedUrlFromAttr(attr pcommon.Map) (string, bool) {
	// this processor enhances url template value, which it extracts from full url or url path.
	// one of these is required for this processor to handle this span.
	urlPath, urlPathFound := getUrlPath(attr)
	if urlPathFound {
		// if url path is available, we can use it to generate the templated url
		// in case of query string, we only want the path part of the url (used with deprecated "http.target" attribute)
		templatedUrl := p.applyTemplatizationOnPath(urlPath)
		return templatedUrl, true
	}

	fullUrl, fullUrlFound := getFullUrl(attr)
	if fullUrlFound {
		parsed, err := url.Parse(fullUrl)
		if err != nil {
			// if we are unable to parse the url, we can't generate the templated url
			// so we skip this span
			return "", false
		}
		templatedUrl := p.applyTemplatizationOnPath(parsed.Path)
		return templatedUrl, true
	}

	return "", false
}

// calculateTemplatedUrlFromAttrWithRules calculates a templated URL using dynamic rules.
func (p *urlTemplateProcessor) calculateTemplatedUrlFromAttrWithRules(attr pcommon.Map, rules map[int][]TemplatizationRule) (string, bool) {
	urlPath, urlPathFound := getUrlPath(attr)
	if urlPathFound {
		templatedUrl := p.applyTemplatizationOnPathWithRules(urlPath, rules, p.customIds)
		return templatedUrl, true
	}

	fullUrl, fullUrlFound := getFullUrl(attr)
	if fullUrlFound {
		parsed, err := url.Parse(fullUrl)
		if err != nil {
			return "", false
		}
		templatedUrl := p.applyTemplatizationOnPathWithRules(parsed.Path, rules, p.customIds)
		return templatedUrl, true
	}

	return "", false
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

func (p *urlTemplateProcessor) enhanceSpan(span ptrace.Span, httpMethod string, targetAttribute string) {

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

	templatedUrl, found := p.calculateTemplatedUrlFromAttr(attr)
	if !found {
		// don't modify the span if we are unable to calculate the templated url
		return
	}

	// set the templated url in the target attribute and update the span name if needed
	attr.PutStr(targetAttribute, templatedUrl)
	updateHttpSpanName(span, httpMethod, templatedUrl)
}

func (p *urlTemplateProcessor) enhanceSpanWithRules(span ptrace.Span, httpMethod string, targetAttribute string, rules map[int][]TemplatizationRule) {
	attr := span.Attributes()

	if val, found := attr.Get(targetAttribute); found {
		if val.Type() != pcommon.ValueTypeStr {
			return
		}
		if val.Str() == "" {
			updateHttpSpanName(span, httpMethod, "/")
		}
		return
	}

	templatedUrl, found := p.calculateTemplatedUrlFromAttrWithRules(attr, rules)
	if !found {
		p.logger.Debug("enhanceSpanWithRules: no url/path in attributes, skip", zap.String("span_name", span.Name()))
		return
	}

	attr.PutStr(targetAttribute, templatedUrl)
	updateHttpSpanName(span, httpMethod, templatedUrl)
}

func (p *urlTemplateProcessor) processSpan(span ptrace.Span) {

	attr := span.Attributes()

	httpMethod, found := getHttpMethod(attr)
	if !found {
		// we only enhance http spans, so if there is no http.method attribute, we can skip it
		return
	}

	switch span.Kind() {

	case ptrace.SpanKindClient:
		// client spans write the url templated value in "url.template" attribute.
		p.enhanceSpan(span, httpMethod, semconv.AttributeURLTemplate)
	case ptrace.SpanKindServer:
		// server spans write the url templated value in "http.route" attribute.
		p.enhanceSpan(span, httpMethod, semconv.AttributeHTTPRoute)
	default:
		// http spans are either client or server
		// all other spans are ignored and never enhanced
		return
	}
}

// processSpanWithRules is like processSpan but uses dynamic rules from the extension cache.
func (p *urlTemplateProcessor) processSpanWithRules(span ptrace.Span, rules map[int][]TemplatizationRule) {
	attr := span.Attributes()

	httpMethod, found := getHttpMethod(attr)
	if !found {
		return
	}

	switch span.Kind() {
	case ptrace.SpanKindClient:
		p.enhanceSpanWithRules(span, httpMethod, semconv.AttributeURLTemplate, rules)
	case ptrace.SpanKindServer:
		p.enhanceSpanWithRules(span, httpMethod, semconv.AttributeHTTPRoute, rules)
	default:
		return
	}
}
