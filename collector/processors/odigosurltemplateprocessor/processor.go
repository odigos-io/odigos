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
)

type urlTemplateProcessor struct {
	logger              *zap.Logger
	templatizationRules []TemplatizationRule
	customIds           []internalCustomIdConfig

	excludeMatcher *PropertiesMatcher
	includeMatcher *PropertiesMatcher
}

func newUrlTemplateProcessor(set processor.Settings, config *Config) (*urlTemplateProcessor, error) {

	excludeMatcher := NewPropertiesMatcher(config.Exclude)
	includeMatcher := NewPropertiesMatcher(config.Include)

	parsedRules := make([]TemplatizationRule, 0, len(config.TemplatizationRules))
	for _, rule := range config.TemplatizationRules {
		parsedRule, err := parseUserInputRuleString(rule)
		if err != nil {
			return nil, err
		}
		parsedRules = append(parsedRules, parsedRule)
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
	}, nil
}

func (p *urlTemplateProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpans := td.ResourceSpans().At(i)

		// before processing the spans, first check if it should be processed according to the include/exclude matchers
		if p.excludeMatcher != nil && p.excludeMatcher.Match(resourceSpans.Resource()) {
			// always skip the resource spans if it matches the exclude matcher
			continue
		}
		// it doesn't make sense to have both include and exclude matchers, but we support it anyway
		if p.includeMatcher != nil && !p.includeMatcher.Match(resourceSpans.Resource()) {
			// if we have an include matcher, it must match the resource for it to be processed
			continue
		}
		// it is ok that both include and exclude matchers are nil, in that case we process all spans

		for j := 0; j < resourceSpans.ScopeSpans().Len(); j++ {
			scopeSpans := resourceSpans.ScopeSpans().At(j)
			for k := 0; k < scopeSpans.Spans().Len(); k++ {
				span := scopeSpans.Spans().At(k)
				p.processSpan(span)
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

func (p *urlTemplateProcessor) applyTemplatizationOnPath(path string) string {
	inputPathSegments := strings.Split(path, "/")
	if len(inputPathSegments) == 0 {
		// if the path is empty, we can't generate a templated url
		return path
	}
	hasLeadingSlash := strings.HasPrefix(path, "/")
	if hasLeadingSlash {
		// if the path has a leading slash, we need to remove it
		// to avoid empty segments in the inputPathSegments
		inputPathSegments = inputPathSegments[1:]
	}

	for _, rule := range p.templatizationRules {
		// apply the rule on the path and return the result if it matches
		if templatedUrl, matched := attemptTemplateWithRule(inputPathSegments, rule); matched {
			if hasLeadingSlash {
				// if the path has a leading slash, we need to add it back
				templatedUrl = "/" + templatedUrl
			}
			return templatedUrl
		}
	}
	templatedPath, isTemplated := defaultTemplatizeURLPath(inputPathSegments, p.customIds)
	if isTemplated {
		if hasLeadingSlash {
			// if the path has a leading slash, we need to add it back
			templatedPath = "/" + templatedPath
		}
		return templatedPath
	} else {
		// if no templated url is generated, we return the original path
		return path
	}
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

	if _, found := attr.Get(targetAttribute); found {
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
