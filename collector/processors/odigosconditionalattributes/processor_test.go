package odigosconditionalattributes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const (
	categoryAttr    = "odigos.category"
	subCategoryAttr = "odigos.sub_category"
	globalDefault   = "Unknown"
)

func newTestProcessor(cfg *Config) *conditionalAttributesProcessor {
	return &conditionalAttributesProcessor{
		logger:              zap.NewNop(),
		config:              cfg,
		uniqueNewAttributes: calculateUniqueNewAttributes(cfg),
	}
}

// scopeSpanInput describes one scope of one resource, so that a test can control the three attribute
// sets the rules are resolved against as well as the instrumentation scope name.
type scopeSpanInput struct {
	scopeName  string
	scopeAttrs map[string]string
	spanAttrs  []map[string]string
}

func newTestTraces(resourceAttrs map[string]string, scopes ...scopeSpanInput) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	putStrings(rs.Resource().Attributes(), resourceAttrs)
	for _, scope := range scopes {
		ss := rs.ScopeSpans().AppendEmpty()
		ss.Scope().SetName(scope.scopeName)
		putStrings(ss.Scope().Attributes(), scope.scopeAttrs)
		for _, spanAttrs := range scope.spanAttrs {
			span := ss.Spans().AppendEmpty()
			putStrings(span.Attributes(), spanAttrs)
		}
	}
	return td
}

func putStrings(target pcommon.Map, attrs map[string]string) {
	for k, v := range attrs {
		target.PutStr(k, v)
	}
}

func spanAttributesAt(t *testing.T, td ptrace.Traces, resource, scope, span int) map[string]any {
	t.Helper()
	require.Greater(t, td.ResourceSpans().Len(), resource)
	rs := td.ResourceSpans().At(resource)
	require.Greater(t, rs.ScopeSpans().Len(), scope)
	ss := rs.ScopeSpans().At(scope)
	require.Greater(t, ss.Spans().Len(), span)
	return ss.Spans().At(span).Attributes().AsRaw()
}

// ****************************************************************
// traces: the documented behaviour of the README examples
// ****************************************************************

func TestProcessTraces_ReadmeExamples(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: OTTLScopeNameKey,
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"opentelemetry.instrumentation.flask": {
						{NewAttributeName: categoryAttr, Value: "flask"},
						{NewAttributeName: subCategoryAttr, Value: "biz"},
					},
				},
			},
			{
				FieldToCheck: "net.host.name",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"coupon": {
						{NewAttributeName: subCategoryAttr, FromField: "http.scheme"},
					},
				},
			},
		},
	}

	tests := []struct {
		name      string
		scopeName string
		spanAttrs map[string]string
		want      map[string]any
	}{
		{
			name:      "matching scope name rule adds both static attributes",
			scopeName: "opentelemetry.instrumentation.flask",
			spanAttrs: map[string]string{"http.scheme": "https"},
			want: map[string]any{
				"http.scheme":   "https",
				categoryAttr:    "flask",
				subCategoryAttr: "biz",
			},
		},
		{
			name:      "partial match copies one attribute and defaults the other",
			scopeName: "",
			spanAttrs: map[string]string{"net.host.name": "coupon", "http.scheme": "https"},
			want: map[string]any{
				"net.host.name": "coupon",
				"http.scheme":   "https",
				categoryAttr:    globalDefault,
				subCategoryAttr: "https",
			},
		},
		{
			name:      "no match defaults every attribute the rules can produce",
			scopeName: "unknown.library",
			spanAttrs: map[string]string{"net.host.name": "unknown"},
			want: map[string]any{
				"net.host.name": "unknown",
				categoryAttr:    globalDefault,
				subCategoryAttr: globalDefault,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := newTestTraces(nil, scopeSpanInput{
				scopeName: tt.scopeName,
				spanAttrs: []map[string]string{tt.spanAttrs},
			})

			out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
			require.NoError(t, err)

			assert.Equal(t, tt.want, spanAttributesAt(t, out, 0, 0, 0))
		})
	}
}

// ****************************************************************
// traces: several source fields for the same new attribute
// ****************************************************************

// The shipped category-attributes profile lists the same new attribute several times with different
// from_field values (db.system, then db.system.name) to cope with semconv renames, and relies on the
// first one that produces a value winning.
func TestProcessTraces_FirstSourceFieldThatExistsWins(t *testing.T) {
	const scopeName = "io.opentelemetry.jedis-4.0"
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: OTTLScopeNameKey,
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					scopeName: {
						{NewAttributeName: categoryAttr, Value: "Database"},
						{NewAttributeName: subCategoryAttr, FromField: "db.system"},
						{NewAttributeName: subCategoryAttr, FromField: "db.system.name"},
					},
				},
			},
		},
	}

	tests := []struct {
		name            string
		spanAttrs       map[string]string
		wantSubCategory string
	}{
		{
			name:            "only the legacy field is present",
			spanAttrs:       map[string]string{"db.system": "redis"},
			wantSubCategory: "redis",
		},
		{
			name:            "only the current field is present",
			spanAttrs:       map[string]string{"db.system.name": "valkey"},
			wantSubCategory: "valkey",
		},
		{
			name:            "both are present so the first configuration wins",
			spanAttrs:       map[string]string{"db.system": "redis", "db.system.name": "valkey"},
			wantSubCategory: "redis",
		},
		{
			name:            "neither is present so the global default applies",
			spanAttrs:       nil,
			wantSubCategory: globalDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := newTestTraces(nil, scopeSpanInput{
				scopeName: scopeName,
				spanAttrs: []map[string]string{tt.spanAttrs},
			})

			out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
			require.NoError(t, err)

			attrs := spanAttributesAt(t, out, 0, 0, 0)
			assert.Equal(t, "Database", attrs[categoryAttr])
			assert.Equal(t, tt.wantSubCategory, attrs[subCategoryAttr])
		})
	}
}

func TestProcessTraces_ScopeNameRuleKeepsAnAttributeTheSpanAlreadyHas(t *testing.T) {
	const scopeName = "opentelemetry.instrumentation.flask"
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: OTTLScopeNameKey,
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					scopeName: {
						{NewAttributeName: categoryAttr, Value: "HTTP"},
						{NewAttributeName: subCategoryAttr, FromField: "http.request.method"},
					},
				},
			},
		},
	}

	td := newTestTraces(nil, scopeSpanInput{
		scopeName: scopeName,
		spanAttrs: []map[string]string{{
			categoryAttr:          "already-categorised",
			subCategoryAttr:       "already-sub-categorised",
			"http.request.method": "GET",
		}},
	})

	out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		categoryAttr:          "already-categorised",
		subCategoryAttr:       "already-sub-categorised",
		"http.request.method": "GET",
	}, spanAttributesAt(t, out, 0, 0, 0))
}

// ****************************************************************
// traces: rules on a regular attribute
// ****************************************************************

func TestProcessTraces_AttributeRuleReadsSpanThenScopeThenResource(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: "deployment.environment",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"production": {{NewAttributeName: categoryAttr, Value: "prod"}},
					"staging":    {{NewAttributeName: categoryAttr, Value: "stg"}},
				},
			},
		},
	}

	tests := []struct {
		name          string
		resourceAttrs map[string]string
		scopeAttrs    map[string]string
		spanAttrs     map[string]string
		wantCategory  string
	}{
		{
			name:         "from the span attributes",
			spanAttrs:    map[string]string{"deployment.environment": "production"},
			wantCategory: "prod",
		},
		{
			name:         "from the scope attributes",
			scopeAttrs:   map[string]string{"deployment.environment": "production"},
			wantCategory: "prod",
		},
		{
			name:          "from the resource attributes",
			resourceAttrs: map[string]string{"deployment.environment": "production"},
			wantCategory:  "prod",
		},
		{
			name:          "the span wins over the scope and the resource",
			resourceAttrs: map[string]string{"deployment.environment": "staging"},
			scopeAttrs:    map[string]string{"deployment.environment": "staging"},
			spanAttrs:     map[string]string{"deployment.environment": "production"},
			wantCategory:  "prod",
		},
		{
			name:          "the scope wins over the resource",
			resourceAttrs: map[string]string{"deployment.environment": "staging"},
			scopeAttrs:    map[string]string{"deployment.environment": "production"},
			wantCategory:  "prod",
		},
		{
			name:          "an unconfigured value falls back to the global default",
			resourceAttrs: map[string]string{"deployment.environment": "dev"},
			wantCategory:  globalDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := newTestTraces(tt.resourceAttrs, scopeSpanInput{
				scopeAttrs: tt.scopeAttrs,
				spanAttrs:  []map[string]string{tt.spanAttrs},
			})

			out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
			require.NoError(t, err)

			assert.Equal(t, tt.wantCategory, spanAttributesAt(t, out, 0, 0, 0)[categoryAttr])
		})
	}
}

// Unlike the instrumentation_scope.name path and the metrics path, which both leave an attribute the
// telemetry already carries untouched, a rule on a regular attribute overwrites the span attribute
// unless the scope and the resource carry it as well.
func TestProcessTraces_AttributeRuleOverwritesAnExistingSpanAttribute(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: "deployment.environment",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"production": {
						{NewAttributeName: categoryAttr, Value: "prod"},
						{NewAttributeName: subCategoryAttr, FromField: "http.scheme"},
					},
				},
			},
		},
	}

	tests := []struct {
		name            string
		resourceAttrs   map[string]string
		scopeAttrs      map[string]string
		wantCategory    string
		wantSubCategory string
	}{
		{
			name:            "only the span carries them",
			wantCategory:    "prod",
			wantSubCategory: "https",
		},
		{
			name: "the span and the scope carry them",
			scopeAttrs: map[string]string{
				categoryAttr:    "set-by-the-scope",
				subCategoryAttr: "set-by-the-scope",
			},
			wantCategory:    "prod",
			wantSubCategory: "https",
		},
		{
			name: "the span, the scope and the resource all carry them",
			scopeAttrs: map[string]string{
				categoryAttr:    "set-by-the-scope",
				subCategoryAttr: "set-by-the-scope",
			},
			resourceAttrs: map[string]string{
				categoryAttr:    "set-by-the-resource",
				subCategoryAttr: "set-by-the-resource",
			},
			wantCategory:    "set-by-the-application",
			wantSubCategory: "set-by-the-application",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := newTestTraces(tt.resourceAttrs, scopeSpanInput{
				scopeAttrs: tt.scopeAttrs,
				spanAttrs: []map[string]string{{
					"deployment.environment": "production",
					"http.scheme":            "https",
					categoryAttr:             "set-by-the-application",
					subCategoryAttr:          "set-by-the-application",
				}},
			})

			out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
			require.NoError(t, err)

			attrs := spanAttributesAt(t, out, 0, 0, 0)
			assert.Equal(t, tt.wantCategory, attrs[categoryAttr])
			assert.Equal(t, tt.wantSubCategory, attrs[subCategoryAttr])
		})
	}
}

func TestProcessTraces_EachSpanIsEvaluatedAgainstItsOwnScope(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: OTTLScopeNameKey,
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"scope.http":     {{NewAttributeName: categoryAttr, Value: "HTTP"}},
					"scope.database": {{NewAttributeName: categoryAttr, Value: "Database"}},
				},
			},
		},
	}

	td := newTestTraces(nil,
		scopeSpanInput{scopeName: "scope.http", spanAttrs: []map[string]string{{}, {}}},
		scopeSpanInput{scopeName: "scope.database", spanAttrs: []map[string]string{{}}},
		scopeSpanInput{scopeName: "scope.unconfigured", spanAttrs: []map[string]string{{}}},
	)

	out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
	require.NoError(t, err)

	assert.Equal(t, "HTTP", spanAttributesAt(t, out, 0, 0, 0)[categoryAttr])
	assert.Equal(t, "HTTP", spanAttributesAt(t, out, 0, 0, 1)[categoryAttr])
	assert.Equal(t, "Database", spanAttributesAt(t, out, 0, 1, 0)[categoryAttr])
	assert.Equal(t, globalDefault, spanAttributesAt(t, out, 0, 2, 0)[categoryAttr])
}

// A scope name rule must read the real instrumentation scope name, not a span attribute that happens
// to be called instrumentation_scope.name.
func TestProcessTraces_ScopeNameRuleIgnoresASpanAttributeOfTheSameName(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: OTTLScopeNameKey,
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"scope.http": {{NewAttributeName: categoryAttr, Value: "HTTP"}},
				},
			},
		},
	}

	td := newTestTraces(nil, scopeSpanInput{
		scopeName: "scope.database",
		spanAttrs: []map[string]string{{OTTLScopeNameKey: "scope.http"}},
	})

	out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		OTTLScopeNameKey: "scope.http",
		categoryAttr:     globalDefault,
	}, spanAttributesAt(t, out, 0, 0, 0))
}

// A configuration keyed by the empty string matches a span whose checked field is missing, because
// the traces path matches the unresolved value, while the metrics path bails out before matching.
func TestConfigurationKeyedByTheEmptyString_AppliesToSpansOnly(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck:        "db.system",
				FieldToCheckMetrics: "db.system",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"": {{NewAttributeName: categoryAttr, Value: "no-db-system"}},
				},
			},
		},
	}

	t.Run("traces", func(t *testing.T) {
		td := newTestTraces(nil, scopeSpanInput{spanAttrs: []map[string]string{{}}})

		out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
		require.NoError(t, err)

		assert.Equal(t, map[string]any{categoryAttr: "no-db-system"}, spanAttributesAt(t, out, 0, 0, 0))
	})

	t.Run("metrics", func(t *testing.T) {
		md := newTestMetrics(pmetric.MetricTypeGauge, nil, map[string]string{})

		out, err := newTestProcessor(cfg).processMetrics(t.Context(), md)
		require.NoError(t, err)

		assert.Equal(t, []map[string]any{{categoryAttr: globalDefault}}, dataPointAttributes(t, out))
	})
}

// Every resource in a batch is evaluated against its own resource attributes: the gateway receives
// spans from many workloads in one request.
func TestProcessTraces_EveryResourceIsEvaluatedAgainstItsOwnAttributes(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: "deployment.environment",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"production": {{NewAttributeName: categoryAttr, Value: "prod"}},
					"staging":    {{NewAttributeName: categoryAttr, Value: "stg"}},
				},
			},
		},
	}

	td := ptrace.NewTraces()
	for _, environment := range []string{"production", "staging", "dev"} {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("deployment.environment", environment)
		rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	}

	out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
	require.NoError(t, err)

	assert.Equal(t, "prod", spanAttributesAt(t, out, 0, 0, 0)[categoryAttr])
	assert.Equal(t, "stg", spanAttributesAt(t, out, 1, 0, 0)[categoryAttr])
	assert.Equal(t, globalDefault, spanAttributesAt(t, out, 2, 0, 0)[categoryAttr])
}

func TestProcessTraces_WithoutRulesSpansAreUntouched(t *testing.T) {
	td := newTestTraces(nil, scopeSpanInput{
		scopeName: "scope.http",
		spanAttrs: []map[string]string{{"http.scheme": "https"}},
	})

	out, err := newTestProcessor(&Config{GlobalDefault: globalDefault}).processTraces(t.Context(), td)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"http.scheme": "https"}, spanAttributesAt(t, out, 0, 0, 0))
}

func TestProcessTraces_ANonStringAttributeIsMatchedByItsStringForm(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck: "http.response.status_code",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"500": {{NewAttributeName: categoryAttr, Value: "server-error"}},
				},
			},
		},
	}

	td := newTestTraces(nil, scopeSpanInput{spanAttrs: []map[string]string{{}}})
	td.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().PutInt("http.response.status_code", 500)

	out, err := newTestProcessor(cfg).processTraces(t.Context(), td)
	require.NoError(t, err)

	assert.Equal(t, "server-error", spanAttributesAt(t, out, 0, 0, 0)[categoryAttr])
}

// ****************************************************************
// metrics
// ****************************************************************

func newTestMetrics(metricType pmetric.MetricType, resourceAttrs map[string]string, dataPointAttrs ...map[string]string) pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	putStrings(rm.Resource().Attributes(), resourceAttrs)
	metric := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
	metric.SetName("test.metric")

	var appendDataPoint func() pcommon.Map
	switch metricType {
	case pmetric.MetricTypeGauge:
		gauge := metric.SetEmptyGauge()
		appendDataPoint = func() pcommon.Map { return gauge.DataPoints().AppendEmpty().Attributes() }
	case pmetric.MetricTypeSum:
		sum := metric.SetEmptySum()
		appendDataPoint = func() pcommon.Map { return sum.DataPoints().AppendEmpty().Attributes() }
	case pmetric.MetricTypeHistogram:
		histogram := metric.SetEmptyHistogram()
		appendDataPoint = func() pcommon.Map { return histogram.DataPoints().AppendEmpty().Attributes() }
	case pmetric.MetricTypeExponentialHistogram:
		expHistogram := metric.SetEmptyExponentialHistogram()
		appendDataPoint = func() pcommon.Map { return expHistogram.DataPoints().AppendEmpty().Attributes() }
	case pmetric.MetricTypeSummary:
		summary := metric.SetEmptySummary()
		appendDataPoint = func() pcommon.Map { return summary.DataPoints().AppendEmpty().Attributes() }
	case pmetric.MetricTypeEmpty:
		return md
	}

	for _, attrs := range dataPointAttrs {
		putStrings(appendDataPoint(), attrs)
	}
	return md
}

func dataPointAttributes(t *testing.T, md pmetric.Metrics) []map[string]any {
	t.Helper()
	require.Equal(t, 1, md.ResourceMetrics().Len())
	metric := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)

	var attrs []pcommon.Map
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		attrs = collectDataPointAttributes(metric.Gauge().DataPoints().Len(), func(i int) pcommon.Map {
			return metric.Gauge().DataPoints().At(i).Attributes()
		})
	case pmetric.MetricTypeSum:
		attrs = collectDataPointAttributes(metric.Sum().DataPoints().Len(), func(i int) pcommon.Map {
			return metric.Sum().DataPoints().At(i).Attributes()
		})
	case pmetric.MetricTypeHistogram:
		attrs = collectDataPointAttributes(metric.Histogram().DataPoints().Len(), func(i int) pcommon.Map {
			return metric.Histogram().DataPoints().At(i).Attributes()
		})
	case pmetric.MetricTypeExponentialHistogram:
		attrs = collectDataPointAttributes(metric.ExponentialHistogram().DataPoints().Len(), func(i int) pcommon.Map {
			return metric.ExponentialHistogram().DataPoints().At(i).Attributes()
		})
	case pmetric.MetricTypeSummary:
		attrs = collectDataPointAttributes(metric.Summary().DataPoints().Len(), func(i int) pcommon.Map {
			return metric.Summary().DataPoints().At(i).Attributes()
		})
	}

	raw := make([]map[string]any, 0, len(attrs))
	for _, attr := range attrs {
		raw = append(raw, attr.AsRaw())
	}
	return raw
}

func collectDataPointAttributes(count int, at func(int) pcommon.Map) []pcommon.Map {
	attrs := make([]pcommon.Map, 0, count)
	for i := 0; i < count; i++ {
		attrs = append(attrs, at(i))
	}
	return attrs
}

func metricsTestConfig() *Config {
	return &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck:        OTTLScopeNameKey,
				FieldToCheckMetrics: "span.instrumentation.scope.name",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"opentelemetry.instrumentation.flask": {
						{NewAttributeName: categoryAttr, Value: "web_framework"},
					},
				},
			},
			{
				FieldToCheck:        "http.method",
				FieldToCheckMetrics: "http.method",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"POST": {{NewAttributeName: subCategoryAttr, Value: "mutation"}},
				},
			},
		},
	}
}

// Every data point of every metric type must be enriched: a type the switch does not handle would
// silently ship data points without the attributes the destination groups by.
func TestProcessMetrics_EveryMetricTypeIsEnriched(t *testing.T) {
	for _, metricType := range []pmetric.MetricType{
		pmetric.MetricTypeGauge,
		pmetric.MetricTypeSum,
		pmetric.MetricTypeHistogram,
		pmetric.MetricTypeExponentialHistogram,
		pmetric.MetricTypeSummary,
	} {
		t.Run(metricType.String(), func(t *testing.T) {
			md := newTestMetrics(metricType, nil,
				map[string]string{"span.instrumentation.scope.name": "opentelemetry.instrumentation.flask", "http.method": "POST"},
				map[string]string{"span.instrumentation.scope.name": "unknown.library", "http.method": "DELETE"},
			)

			out, err := newTestProcessor(metricsTestConfig()).processMetrics(t.Context(), md)
			require.NoError(t, err)

			assert.Equal(t, []map[string]any{
				{
					"span.instrumentation.scope.name": "opentelemetry.instrumentation.flask",
					"http.method":                     "POST",
					categoryAttr:                      "web_framework",
					subCategoryAttr:                   "mutation",
				},
				{
					"span.instrumentation.scope.name": "unknown.library",
					"http.method":                     "DELETE",
					categoryAttr:                      globalDefault,
					subCategoryAttr:                   globalDefault,
				},
			}, dataPointAttributes(t, out))
		})
	}
}

func TestProcessMetrics_AMetricWithoutDataIsLeftAlone(t *testing.T) {
	md := newTestMetrics(pmetric.MetricTypeEmpty, nil, map[string]string{"http.method": "POST"})

	out, err := newTestProcessor(metricsTestConfig()).processMetrics(t.Context(), md)
	require.NoError(t, err)

	assert.Equal(t, pmetric.MetricTypeEmpty, out.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Type())
	assert.Empty(t, dataPointAttributes(t, out))
}

func TestProcessMetrics_RuleWithoutAMetricsFieldIsSkipped(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				// traces-only rule: instrumentation_scope.name has no metrics equivalent.
				FieldToCheck: OTTLScopeNameKey,
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"opentelemetry.instrumentation.flask": {
						{NewAttributeName: categoryAttr, Value: "web_framework"},
					},
				},
			},
		},
	}

	md := newTestMetrics(pmetric.MetricTypeGauge, nil, map[string]string{
		OTTLScopeNameKey: "opentelemetry.instrumentation.flask",
	})

	out, err := newTestProcessor(cfg).processMetrics(t.Context(), md)
	require.NoError(t, err)

	// the rule contributes its new attribute to the defaults even though it never matches a metric.
	assert.Equal(t, []map[string]any{{
		OTTLScopeNameKey: "opentelemetry.instrumentation.flask",
		categoryAttr:     globalDefault,
	}}, dataPointAttributes(t, out))
}

func TestProcessMetrics_DataPointAttributesTakePrecedenceOverTheResource(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck:        "deployment.environment",
				FieldToCheckMetrics: "deployment.environment",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"production": {{NewAttributeName: categoryAttr, Value: "prod"}},
					"staging":    {{NewAttributeName: categoryAttr, Value: "stg"}},
				},
			},
		},
	}

	t.Run("resolved from the resource", func(t *testing.T) {
		md := newTestMetrics(pmetric.MetricTypeSum,
			map[string]string{"deployment.environment": "production"},
			map[string]string{},
		)

		out, err := newTestProcessor(cfg).processMetrics(t.Context(), md)
		require.NoError(t, err)

		assert.Equal(t, []map[string]any{{categoryAttr: "prod"}}, dataPointAttributes(t, out))
	})

	t.Run("the data point wins over the resource", func(t *testing.T) {
		md := newTestMetrics(pmetric.MetricTypeSum,
			map[string]string{"deployment.environment": "staging"},
			map[string]string{"deployment.environment": "production"},
		)

		out, err := newTestProcessor(cfg).processMetrics(t.Context(), md)
		require.NoError(t, err)

		assert.Equal(t, []map[string]any{{
			"deployment.environment": "production",
			categoryAttr:             "prod",
		}}, dataPointAttributes(t, out))
	})
}

func TestProcessMetrics_CopiesFromAnotherDataPointAttribute(t *testing.T) {
	cfg := &Config{
		GlobalDefault: globalDefault,
		Rules: []ConditionalRule{
			{
				FieldToCheck:        "db.system",
				FieldToCheckMetrics: "db.system",
				NewAttributeValueConfigurations: map[string][]NewAttributeValueConfiguration{
					"postgresql": {
						{NewAttributeName: categoryAttr, Value: "Database"},
						{NewAttributeName: subCategoryAttr, FromField: "db.namespace"},
					},
				},
			},
		},
	}

	t.Run("the source attribute exists", func(t *testing.T) {
		md := newTestMetrics(pmetric.MetricTypeGauge, nil, map[string]string{
			"db.system":    "postgresql",
			"db.namespace": "orders",
		})

		out, err := newTestProcessor(cfg).processMetrics(t.Context(), md)
		require.NoError(t, err)

		assert.Equal(t, []map[string]any{{
			"db.system":    "postgresql",
			"db.namespace": "orders",
			categoryAttr:   "Database",
			// the copy is taken from the data point, resource attributes are not consulted.
			subCategoryAttr: "orders",
		}}, dataPointAttributes(t, out))
	})

	t.Run("the target attribute already exists", func(t *testing.T) {
		md := newTestMetrics(pmetric.MetricTypeGauge, nil, map[string]string{
			"db.system":     "postgresql",
			"db.namespace":  "orders",
			subCategoryAttr: "set-by-the-application",
		})

		out, err := newTestProcessor(cfg).processMetrics(t.Context(), md)
		require.NoError(t, err)

		assert.Equal(t, []map[string]any{{
			"db.system":     "postgresql",
			"db.namespace":  "orders",
			categoryAttr:    "Database",
			subCategoryAttr: "set-by-the-application",
		}}, dataPointAttributes(t, out))
	})

	t.Run("the source attribute is only on the resource", func(t *testing.T) {
		md := newTestMetrics(pmetric.MetricTypeGauge,
			map[string]string{"db.namespace": "orders"},
			map[string]string{"db.system": "postgresql"},
		)

		out, err := newTestProcessor(cfg).processMetrics(t.Context(), md)
		require.NoError(t, err)

		assert.Equal(t, []map[string]any{{
			"db.system":     "postgresql",
			categoryAttr:    "Database",
			subCategoryAttr: globalDefault,
		}}, dataPointAttributes(t, out))
	})
}

func TestProcessMetrics_KeepsAnAttributeTheDataPointAlreadyHas(t *testing.T) {
	md := newTestMetrics(pmetric.MetricTypeGauge, nil, map[string]string{
		"http.method":   "POST",
		subCategoryAttr: "set-by-the-application",
	})

	out, err := newTestProcessor(metricsTestConfig()).processMetrics(t.Context(), md)
	require.NoError(t, err)

	assert.Equal(t, []map[string]any{{
		"http.method":   "POST",
		subCategoryAttr: "set-by-the-application",
		categoryAttr:    globalDefault,
	}}, dataPointAttributes(t, out))
}

func TestProcessMetrics_EveryResourceScopeAndMetricIsVisited(t *testing.T) {
	md := pmetric.NewMetrics()
	for _, method := range []string{"POST", "GET"} {
		rm := md.ResourceMetrics().AppendEmpty()
		for range 2 {
			sm := rm.ScopeMetrics().AppendEmpty()
			for _, name := range []string{"first.metric", "second.metric"} {
				metric := sm.Metrics().AppendEmpty()
				metric.SetName(name)
				metric.SetEmptyGauge().DataPoints().AppendEmpty().Attributes().PutStr("http.method", method)
			}
		}
	}

	out, err := newTestProcessor(metricsTestConfig()).processMetrics(t.Context(), md)
	require.NoError(t, err)

	resources := out.ResourceMetrics()
	require.Equal(t, 2, resources.Len())
	for r := 0; r < resources.Len(); r++ {
		scopes := resources.At(r).ScopeMetrics()
		require.Equal(t, 2, scopes.Len())
		for i := 0; i < scopes.Len(); i++ {
			metrics := scopes.At(i).Metrics()
			require.Equal(t, 2, metrics.Len())
			for j := 0; j < metrics.Len(); j++ {
				attrs := metrics.At(j).Gauge().DataPoints().At(0).Attributes().AsRaw()
				wantSubCategory := globalDefault
				if attrs["http.method"] == "POST" {
					wantSubCategory = "mutation"
				}
				assert.Equal(t, wantSubCategory, attrs[subCategoryAttr], "resource %d scope %d metric %d", r, i, j)
				assert.Equal(t, globalDefault, attrs[categoryAttr], "resource %d scope %d metric %d", r, i, j)
			}
		}
	}
}

// A metric that carries neither the checked attribute nor a resource level one still gets the
// defaults, so the destination always sees the same attribute set.
func TestProcessMetrics_MissingCheckedAttributeOnlyGetsTheDefaults(t *testing.T) {
	md := newTestMetrics(pmetric.MetricTypeGauge, nil, map[string]string{"unrelated": "value"})

	out, err := newTestProcessor(metricsTestConfig()).processMetrics(t.Context(), md)
	require.NoError(t, err)

	assert.Equal(t, []map[string]any{{
		"unrelated":     "value",
		categoryAttr:    globalDefault,
		subCategoryAttr: globalDefault,
	}}, dataPointAttributes(t, out))
}
