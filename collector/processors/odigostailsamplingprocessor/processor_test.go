package odigostailsamplingprocessor

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

// The processor derives the trace's position in the [0,100) keep window from the least significant
// 7 bytes of the trace ID. Powers of two keep the scaling exact in float64, so a rule percentage
// compared against these values is not subject to rounding.
const (
	randomnessZeroPercent       = uint64(0)
	randomnessTwentyFivePercent = uint64(1) << 54
	randomnessFiftyPercent      = uint64(1) << 55
)

const (
	samplingTestWorkloadKeyAttr = "odigos.test.workload.key"
	samplingTestWorkloadKey     = "default/deployment/checkout/app"
)

// samplingTraceID builds a trace ID whose sampling randomness is exactly the given value.
func samplingTraceID(randomness uint64) pcommon.TraceID {
	var id pcommon.TraceID
	binary.BigEndian.PutUint64(id[8:], randomness)
	return id
}

// samplingTestExtension resolves the workload cache key from a resource attribute so a single
// trace can carry resources belonging to different workloads.
type samplingTestExtension struct {
	callbacks []collector.WorkloadConfigCacheCallback
	synced    bool
}

func (s *samplingTestExtension) Start(context.Context, component.Host) error { return nil }
func (s *samplingTestExtension) Shutdown(context.Context) error              { return nil }

func (s *samplingTestExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (s *samplingTestExtension) IsActiveSource(pcommon.Resource) bool { return true }

func (s *samplingTestExtension) GetWorkloadCacheKey(res pcommon.Resource) (string, error) {
	key, found := res.Attributes().Get(samplingTestWorkloadKeyAttr)
	if !found {
		return "", assert.AnError
	}
	return key.Str(), nil
}

func (s *samplingTestExtension) GetWorkloadIdentityFromResource(res pcommon.Resource) (string, pcommon.Map, error) {
	key, err := s.GetWorkloadCacheKey(res)
	return key, pcommon.NewMap(), err
}

func (s *samplingTestExtension) RegisterWorkloadConfigCacheCallback(cb collector.WorkloadConfigCacheCallback) {
	s.callbacks = append(s.callbacks, cb)
}

func (s *samplingTestExtension) UnregisterWorkloadConfigCacheCallback(cb collector.WorkloadConfigCacheCallback) {
	for i, registered := range s.callbacks {
		if registered == cb {
			s.callbacks = append(s.callbacks[:i], s.callbacks[i+1:]...)
			return
		}
	}
}

func (s *samplingTestExtension) WaitForCacheSync(context.Context) bool { return s.synced }

func (s *samplingTestExtension) GetDataStreamsForWorkload(pcommon.Resource) ([]string, bool) {
	return nil, false
}

type samplingTestHost struct {
	extensions map[component.ID]component.Component
}

func (h samplingTestHost) GetExtensions() map[component.ID]component.Component { return h.extensions }

// samplingTestEnv is a started processor wired to a stub odigos config extension, plus the
// in-memory telemetry the processor records its sampling metrics to.
type samplingTestEnv struct {
	proc      *tailSamplingProcessor
	telemetry *componenttest.Telemetry
}

// newSamplingTestEnv builds and starts a processor with the given config, and attaches it to a
// stub odigos config extension so the workload config cache is live.
func newSamplingTestEnv(t *testing.T, cfg *Config) *samplingTestEnv {
	t.Helper()

	extID := component.MustNewID("odigosconfigk8s")
	cfg.OdigosConfigExtension = &extID

	tt := componenttest.NewTelemetry()
	t.Cleanup(func() {
		require.NoError(t, tt.Shutdown(context.Background()))
	})

	proc := newTailSamplingProcessor(zap.NewNop(), cfg, tt.NewTelemetrySettings())
	host := samplingTestHost{extensions: map[component.ID]component.Component{
		extID: &samplingTestExtension{synced: true},
	}}
	require.NoError(t, proc.Start(context.Background(), host))
	t.Cleanup(func() {
		require.NoError(t, proc.Shutdown(context.Background()))
	})

	return &samplingTestEnv{proc: proc, telemetry: tt}
}

// setWorkloadConfig pushes a tail sampling config for a workload key through the same callback the
// odigos config extension uses in production, so the precompute step is exercised too.
func (e *samplingTestEnv) setWorkloadConfig(key string, cfg *commonapisampling.TailSamplingSourceConfig) {
	e.proc.configCache.OnSet(key, &commonapi.ContainerCollectorConfig{
		ContainerName: "app",
		TailSampling:  cfg,
	})
}

func (e *samplingTestEnv) process(t *testing.T, td ptrace.Traces) ptrace.Traces {
	t.Helper()
	out, err := e.proc.processTraces(context.Background(), td)
	require.NoError(t, err)
	return out
}

// sumMetricForCategory returns the total of an int64 sum metric across the data points whose
// odigos.sampling.category attribute equals the given category, ignoring per-rule data points.
func (e *samplingTestEnv) sumMetricForCategory(t *testing.T, metricName string, category string) int64 {
	t.Helper()
	metric, err := e.telemetry.GetMetric(metricName)
	if err != nil {
		return 0
	}
	sum, ok := metric.Data.(metricdata.Sum[int64])
	require.True(t, ok, "metric %q is not an int64 sum", metricName)

	var total int64
	for _, dp := range sum.DataPoints {
		value, found := dp.Attributes.Value(odigosattributes.SamplingCategory)
		if !found || value.AsString() != category {
			continue
		}
		// per-rule data points also carry the category; skip them so only the category level
		// aggregate is counted.
		if _, isRuleDataPoint := dp.Attributes.Value(odigosattributes.SamplingRuleId); isRuleDataPoint {
			continue
		}
		total += dp.Value
	}
	return total
}

// sumMetricForRule returns the total of an int64 sum metric across the data points belonging to a
// single rule.
func (e *samplingTestEnv) sumMetricForRule(t *testing.T, metricName string, ruleID string) int64 {
	t.Helper()
	metric, err := e.telemetry.GetMetric(metricName)
	if err != nil {
		return 0
	}
	sum, ok := metric.Data.(metricdata.Sum[int64])
	require.True(t, ok, "metric %q is not an int64 sum", metricName)

	var total int64
	for _, dp := range sum.DataPoints {
		value, found := dp.Attributes.Value(odigosattributes.SamplingRuleId)
		if !found || value.AsString() != ruleID {
			continue
		}
		total += dp.Value
	}
	return total
}

type samplingTestSpanOpt func(ptrace.Span)

func asChildSpan() samplingTestSpanOpt {
	return func(span ptrace.Span) { span.SetParentSpanID(pcommon.SpanID{1, 2, 3, 4, 5, 6, 7, 8}) }
}

func withErrorStatus() samplingTestSpanOpt {
	return func(span ptrace.Span) { span.Status().SetCode(ptrace.StatusCodeError) }
}

func withHeadSampledTraceState() samplingTestSpanOpt {
	return func(span ptrace.Span) { span.TraceState().FromRaw("odigos=head") }
}

// appendHttpServerSpan adds an http server span for the given route, which is what both the head
// sampling (noisy) and tail sampling (highly relevant / cost reduction) http server matchers need.
func appendHttpServerSpan(rs ptrace.ResourceSpans, traceID pcommon.TraceID, route string, opts ...samplingTestSpanOpt) ptrace.Span {
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(traceID)
	span.SetName("GET " + route)
	span.SetKind(ptrace.SpanKindServer)
	span.Attributes().PutStr("http.request.method", "GET")
	span.Attributes().PutStr("http.route", route)
	for _, opt := range opts {
		opt(span)
	}
	return span
}

// newSamplingTestTrace builds a single-resource trace for samplingTestWorkloadKey holding one root
// http server span on the given route.
func newSamplingTestTrace(randomness uint64, route string, opts ...samplingTestSpanOpt) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr(samplingTestWorkloadKeyAttr, samplingTestWorkloadKey)
	appendHttpServerSpan(rs, samplingTraceID(randomness), route, opts...)
	return td
}

func firstSpanOf(t *testing.T, td ptrace.Traces) ptrace.Span {
	t.Helper()
	require.Positive(t, td.ResourceSpans().Len())
	return td.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
}

func spanAttrStr(t *testing.T, span ptrace.Span, key string) string {
	t.Helper()
	value, found := span.Attributes().Get(key)
	require.True(t, found, "span is missing attribute %q", key)
	return value.Str()
}

func float64Ptr(f float64) *float64 { return &f }

func noisyRule(id string, route string, percentageAtMost *float64) commonapisampling.NoisyOperation {
	return commonapisampling.NoisyOperation{
		Id:   id,
		Name: id,
		Operation: &commonapisampling.HeadSamplingOperationMatcher{
			HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: route},
		},
		PercentageAtMost: percentageAtMost,
	}
}

func highlyRelevantRule(id string, route string, percentageAtLeast *float64) commonapisampling.HighlyRelevantOperation {
	return commonapisampling.HighlyRelevantOperation{
		Id:   id,
		Name: id,
		Operation: &commonapisampling.TailSamplingOperationMatcher{
			HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: route},
		},
		PercentageAtLeast: percentageAtLeast,
	}
}

func costReductionRule(id string, route string, percentageAtMost float64) commonapisampling.CostReductionRule {
	return commonapisampling.CostReductionRule{
		Id:   id,
		Name: id,
		Operation: &commonapisampling.TailSamplingOperationMatcher{
			HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: route},
		},
		PercentageAtMost: percentageAtMost,
	}
}

func TestConfigValidate(t *testing.T) {
	err := (&Config{}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "odigos config extension is required")

	extID := component.MustNewID("odigosconfigk8s")
	assert.NoError(t, (&Config{OdigosConfigExtension: &extID}).Validate())
}

// Without the odigos config extension the processor has no rules at all, so it must forward the
// trace. Dropping here would silently discard every trace in the pipeline.
func TestProcessTraces_ForwardsEverythingWhenNotAttachedToTheExtension(t *testing.T) {
	proc := newTailSamplingProcessor(zap.NewNop(), &Config{}, componenttest.NewNopTelemetrySettings())

	td := newSamplingTestTrace(randomnessFiftyPercent, "/checkout")
	out, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)
	assert.Equal(t, 1, out.SpanCount())
}

// A workload with no tail sampling config must never be sampled, otherwise enabling sampling for
// one source would start dropping traces of every other source in the cluster.
func TestProcessTraces_ForwardsTraceOfWorkloadWithoutConfig(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig("default/deployment/other/app", &commonapisampling.TailSamplingSourceConfig{
		NoisyOperations: []commonapisampling.NoisyOperation{noisyRule("noise-1", "/checkout", nil)},
	})

	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout"))
	assert.Equal(t, 1, out.SpanCount())
	_, hasCategory := firstSpanOf(t, out).Attributes().Get(odigosattributes.SamplingCategory)
	assert.False(t, hasCategory)
}

// Head sampling already applied a probability to this trace. Re-applying tail sampling would
// multiply the two and drop far more than the user configured.
func TestProcessTraces_SkipsTraceAlreadyDecidedByHeadSampling(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		// 0% - would drop the trace if tail sampling ran.
		NoisyOperations: []commonapisampling.NoisyOperation{noisyRule("noise-1", "/checkout", nil)},
	})

	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout", withHeadSampledTraceState()))
	require.Equal(t, 1, out.SpanCount())
	// no category was evaluated at all, so the span comes out exactly as it went in.
	assert.Equal(t, map[string]any{
		"http.request.method": "GET",
		"http.route":          "/checkout",
	}, firstSpanOf(t, out).Attributes().AsRaw())
}

// A batch holding more than one trace means this processor is not running after groupbytraceid.
// It must bail out by forwarding the batch, never by dropping it.
func TestProcessTraces_ForwardsBatchWithMultipleTraceIds(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		NoisyOperations: []commonapisampling.NoisyOperation{noisyRule("noise-1", "/checkout", nil)},
	})

	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr(samplingTestWorkloadKeyAttr, samplingTestWorkloadKey)
	appendHttpServerSpan(rs, samplingTraceID(randomnessZeroPercent), "/checkout")
	appendHttpServerSpan(rs, samplingTraceID(randomnessFiftyPercent), "/checkout")

	out := env.process(t, td)
	assert.Equal(t, 2, out.SpanCount())
}

func TestProcessTraces_ForwardsTraceWhenNoRuleMatches(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		NoisyOperations:          []commonapisampling.NoisyOperation{noisyRule("noise-1", "/healthz", nil)},
		HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{highlyRelevantRule("hr-1", "/healthz", nil)},
		CostReductionRules:       []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/healthz", 0)},
	})

	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout"))
	require.Equal(t, 1, out.SpanCount())
	_, hasCategory := firstSpanOf(t, out).Attributes().Get(odigosattributes.SamplingCategory)
	assert.False(t, hasCategory)
}

// The three categories are strictly ordered: noise decides before highly relevant, which decides
// before cost reduction. A reordering would let a "keep everything" highly relevant rule override
// an explicit noise rule (blowing up cost) or let a cost reduction rule drop a trace the user
// marked as highly relevant (losing the traces that matter most).
func TestProcessTraces_CategoryPrecedence(t *testing.T) {
	allCategoriesConfig := func(noisePercentage *float64, highlyRelevantPercentage *float64, costPercentage float64) *commonapisampling.TailSamplingSourceConfig {
		return &commonapisampling.TailSamplingSourceConfig{
			NoisyOperations:          []commonapisampling.NoisyOperation{noisyRule("noise-1", "/checkout", noisePercentage)},
			HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{highlyRelevantRule("hr-1", "/checkout", highlyRelevantPercentage)},
			CostReductionRules:       []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", costPercentage)},
		}
	}

	tests := []struct {
		name             string
		config           *commonapisampling.TailSamplingSourceConfig
		wantKept         bool
		wantCategory     string
		wantDecidingRule string
	}{
		{
			name:             "noise decides even when highly relevant would keep the trace",
			config:           allCategoriesConfig(float64Ptr(25), float64Ptr(100), 100),
			wantKept:         false,
			wantCategory:     "noise",
			wantDecidingRule: "noise-1",
		},
		{
			name:             "noise decides to keep even when cost reduction would drop the trace",
			config:           allCategoriesConfig(float64Ptr(75), float64Ptr(0), 0),
			wantKept:         true,
			wantCategory:     "noise",
			wantDecidingRule: "noise-1",
		},
		{
			name: "highly relevant decides when no noise rule matches",
			config: &commonapisampling.TailSamplingSourceConfig{
				HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{highlyRelevantRule("hr-1", "/checkout", float64Ptr(75))},
				CostReductionRules:       []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 0)},
			},
			wantKept:         true,
			wantCategory:     "highly relevant",
			wantDecidingRule: "hr-1",
		},
		{
			name: "cost reduction decides only when neither other category matches",
			config: &commonapisampling.TailSamplingSourceConfig{
				NoisyOperations:          []commonapisampling.NoisyOperation{noisyRule("noise-1", "/healthz", nil)},
				HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{highlyRelevantRule("hr-1", "/healthz", nil)},
				CostReductionRules:       []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 75)},
			},
			wantKept:         true,
			wantCategory:     "cost reduction",
			wantDecidingRule: "cost-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newSamplingTestEnv(t, &Config{})
			env.setWorkloadConfig(samplingTestWorkloadKey, tt.config)

			out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout"))

			if !tt.wantKept {
				assert.Zero(t, out.SpanCount())
				return
			}
			require.Equal(t, 1, out.SpanCount())
			span := firstSpanOf(t, out)
			assert.Equal(t, tt.wantCategory, spanAttrStr(t, span, odigosattributes.SamplingCategory))
			assert.Equal(t, tt.wantDecidingRule, spanAttrStr(t, span, odigosattributes.SamplingTraceDecidingRuleId))
		})
	}
}

// The trace is kept when its position in the keep window is at or below the deciding rule's
// percentage. Each of the three categories compares the window on its own, so the boundary is
// checked for each: an off-by-one there silently changes the sampling rate of every rule in that
// category.
func TestProcessTraces_KeepWindowIsInclusive(t *testing.T) {
	categories := map[string]func(percentage float64) *commonapisampling.TailSamplingSourceConfig{
		"noise": func(percentage float64) *commonapisampling.TailSamplingSourceConfig {
			return &commonapisampling.TailSamplingSourceConfig{
				NoisyOperations: []commonapisampling.NoisyOperation{noisyRule("noise-1", "/checkout", float64Ptr(percentage))},
			}
		},
		"highly relevant": func(percentage float64) *commonapisampling.TailSamplingSourceConfig {
			return &commonapisampling.TailSamplingSourceConfig{
				HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{highlyRelevantRule("hr-1", "/checkout", float64Ptr(percentage))},
			}
		},
		"cost reduction": func(percentage float64) *commonapisampling.TailSamplingSourceConfig {
			return &commonapisampling.TailSamplingSourceConfig{
				CostReductionRules: []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", percentage)},
			}
		},
	}

	tests := []struct {
		name             string
		randomness       uint64
		rulePercentage   float64
		wantSpansForward int
	}{
		{
			name:             "trace below the rule percentage is kept",
			randomness:       randomnessTwentyFivePercent,
			rulePercentage:   50,
			wantSpansForward: 1,
		},
		{
			name:             "trace exactly at the rule percentage is kept",
			randomness:       randomnessFiftyPercent,
			rulePercentage:   50,
			wantSpansForward: 1,
		},
		{
			name:             "trace just above the rule percentage is dropped",
			randomness:       randomnessFiftyPercent,
			rulePercentage:   49.9,
			wantSpansForward: 0,
		},
		{
			name:             "trace well above the rule percentage is dropped",
			randomness:       randomnessFiftyPercent,
			rulePercentage:   25,
			wantSpansForward: 0,
		},
		{
			// a 0% rule is not "drop everything": the single trace sitting exactly at the bottom of
			// the window is still kept, because the comparison is inclusive.
			name:             "a zero percent rule keeps the trace at the bottom of the window",
			randomness:       randomnessZeroPercent,
			rulePercentage:   0,
			wantSpansForward: 1,
		},
		{
			name:             "a zero percent rule drops any trace above the bottom of the window",
			randomness:       randomnessTwentyFivePercent,
			rulePercentage:   0,
			wantSpansForward: 0,
		},
	}

	for category, configFor := range categories {
		for _, tt := range tests {
			t.Run(category+"/"+tt.name, func(t *testing.T) {
				env := newSamplingTestEnv(t, &Config{})
				env.setWorkloadConfig(samplingTestWorkloadKey, configFor(tt.rulePercentage))

				out := env.process(t, newSamplingTestTrace(tt.randomness, "/checkout"))
				assert.Equal(t, tt.wantSpansForward, out.SpanCount())
			})
		}
	}
}

// Every category can also decide to drop. A highly relevant rule is a floor on the keep
// percentage, not an unconditional keep, so a trace above that floor is dropped even though the
// category matched.
func TestProcessTraces_HighlyRelevantCanDropTheTrace(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{highlyRelevantRule("hr-1", "/checkout", float64Ptr(25))},
		// a cost reduction rule that would keep the trace must not be reached.
		CostReductionRules: []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 100)},
	})

	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout"))
	assert.Zero(t, out.SpanCount())

	assert.Equal(t, int64(1), env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.drop_count", "highly relevant"))
	assert.Zero(t, env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.match_count", "cost reduction"))
}

// Dry run exists so users can evaluate rules before committing to them. It must never drop a
// trace, and it must record what the decision would have been.
func TestProcessTraces_DryRunForwardsTheTraceItWouldHaveDropped(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{DryRun: true})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		CostReductionRules: []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 25)},
	})

	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout"))
	require.Equal(t, 1, out.SpanCount())

	span := firstSpanOf(t, out)
	assert.Equal(t, "cost reduction", spanAttrStr(t, span, odigosattributes.SamplingCategory))

	dryRun, found := span.Attributes().Get(odigosattributes.SamplingDryRun)
	require.True(t, found)
	assert.True(t, dryRun.Bool())

	kept, found := span.Attributes().Get(odigosattributes.SamplingTraceKept)
	require.True(t, found)
	assert.False(t, kept.Bool(), "dry run must record that the trace would have been dropped")

	// the decision is still counted, so dry run dashboards show the effect of enabling the rules.
	assert.Equal(t, int64(1), env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.drop_count", "cost reduction"))
	assert.Zero(t, env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.keep_count", "cost reduction"))
}

func TestProcessTraces_DryRunMarksTheTraceItWouldHaveKept(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{DryRun: true})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		CostReductionRules: []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 75)},
	})

	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout"))
	require.Equal(t, 1, out.SpanCount())

	kept, found := firstSpanOf(t, out).Attributes().Get(odigosattributes.SamplingTraceKept)
	require.True(t, found)
	assert.True(t, kept.Bool())
}

// Noisy rules are head-sampling matchers, evaluated on the root span only. A trace fragment that
// arrives without its root span must not be judged by the noise category, or an unrelated span on
// the same route would drop a trace the user never marked as noisy.
func TestProcessTraces_NoiseCategoryNeedsTheRootSpan(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		NoisyOperations:    []commonapisampling.NoisyOperation{noisyRule("noise-1", "/checkout", float64Ptr(25))},
		CostReductionRules: []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 75)},
	})

	// the only span carries a parent, so the trace has no root span.
	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout", asChildSpan()))
	require.Equal(t, 1, out.SpanCount())
	assert.Equal(t, "cost reduction", spanAttrStr(t, firstSpanOf(t, out), odigosattributes.SamplingCategory))
}

// Highly relevant and cost reduction rules are evaluated on every span, not just the root, so a
// deep error span must be able to rescue the whole trace.
func TestProcessTraces_HighlyRelevantMatchesANonRootSpan(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		HighlyRelevantOperations: []commonapisampling.HighlyRelevantOperation{{
			Id:                "hr-errors",
			Name:              "hr-errors",
			Error:             true,
			PercentageAtLeast: float64Ptr(100),
		}},
		CostReductionRules: []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 25)},
	})

	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr(samplingTestWorkloadKeyAttr, samplingTestWorkloadKey)
	traceID := samplingTraceID(randomnessFiftyPercent)
	appendHttpServerSpan(rs, traceID, "/checkout")
	appendHttpServerSpan(rs, traceID, "/checkout", asChildSpan(), withErrorStatus())

	out := env.process(t, td)
	require.Equal(t, 2, out.SpanCount())
	assert.Equal(t, "highly relevant", spanAttrStr(t, firstSpanOf(t, out), odigosattributes.SamplingCategory))
}

// Only resources whose workload has rules are evaluated. A trace spanning an instrumented and an
// uninstrumented workload must still be decided by the rules of the instrumented one.
func TestProcessTraces_EvaluatesOnlyResourcesWithConfig(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		CostReductionRules: []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 25)},
	})

	td := ptrace.NewTraces()
	traceID := samplingTraceID(randomnessFiftyPercent)

	unconfigured := td.ResourceSpans().AppendEmpty()
	unconfigured.Resource().Attributes().PutStr(samplingTestWorkloadKeyAttr, "default/deployment/frontend/app")
	appendHttpServerSpan(unconfigured, traceID, "/checkout")

	configured := td.ResourceSpans().AppendEmpty()
	configured.Resource().Attributes().PutStr(samplingTestWorkloadKeyAttr, samplingTestWorkloadKey)
	appendHttpServerSpan(configured, traceID, "/checkout", asChildSpan())

	out := env.process(t, td)
	assert.Zero(t, out.SpanCount(), "the configured workload's 25%% rule must drop the whole trace")
}

func TestProcessTraces_RecordsCategoryAndRuleMetrics(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		CostReductionRules: []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 25)},
	})

	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout"))
	require.Zero(t, out.SpanCount())

	assert.Equal(t, int64(1), env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.match_count", "cost reduction"))
	assert.Equal(t, int64(1), env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.drop_count", "cost reduction"))
	assert.Zero(t, env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.keep_count", "cost reduction"))

	// the rule that matched is counted as dropping the trace on its own series, which is what the
	// per-rule sampling dashboards read.
	assert.Equal(t, int64(1), env.sumMetricForRule(t, "otelcol_odigos.sampling.trace.drop_count", "cost-1"))
	assert.Zero(t, env.sumMetricForRule(t, "otelcol_odigos.sampling.trace.keep_count", "cost-1"))
	assert.Equal(t, int64(1), env.sumMetricForRule(t, "otelcol_odigos.sampling.trace.match_count", "cost-1"))

	// nothing matched the other two categories, so they must not report a decision.
	for _, category := range []string{"noise", "highly relevant"} {
		assert.Zero(t, env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.match_count", category), category)
		assert.Zero(t, env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.drop_count", category), category)
		assert.Zero(t, env.sumMetricForCategory(t, "otelcol_odigos.sampling.trace.keep_count", category), category)
	}
}

// A rule the trace stays within is counted as keeping it, the mirror of the drop case above.
func TestProcessTraces_RecordsPerRuleKeepMetrics(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		CostReductionRules: []commonapisampling.CostReductionRule{costReductionRule("cost-1", "/checkout", 75)},
	})

	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout"))
	require.Equal(t, 1, out.SpanCount())

	assert.Equal(t, int64(1), env.sumMetricForRule(t, "otelcol_odigos.sampling.trace.keep_count", "cost-1"))
	assert.Zero(t, env.sumMetricForRule(t, "otelcol_odigos.sampling.trace.drop_count", "cost-1"))
}

// A disabled rule must be reported in the metrics (so users can see what it would do) but must
// never influence the decision.
func TestProcessTraces_DisabledRuleIsMeasuredButDoesNotDecide(t *testing.T) {
	env := newSamplingTestEnv(t, &Config{})
	disabled := costReductionRule("cost-disabled", "/checkout", 0)
	disabled.Disabled = true
	env.setWorkloadConfig(samplingTestWorkloadKey, &commonapisampling.TailSamplingSourceConfig{
		CostReductionRules: []commonapisampling.CostReductionRule{disabled},
	})

	out := env.process(t, newSamplingTestTrace(randomnessFiftyPercent, "/checkout"))
	require.Equal(t, 1, out.SpanCount(), "a disabled rule must not drop the trace")

	_, hasCategory := firstSpanOf(t, out).Attributes().Get(odigosattributes.SamplingCategory)
	assert.False(t, hasCategory)

	metric, err := env.telemetry.GetMetric("otelcol_odigos.sampling.trace.match_count")
	require.NoError(t, err)
	sum, ok := metric.Data.(metricdata.Sum[int64])
	require.True(t, ok)

	var disabledRuleMatches int64
	for _, dp := range sum.DataPoints {
		ruleID, found := dp.Attributes.Value(odigosattributes.SamplingRuleId)
		if !found || ruleID.AsString() != "cost-disabled" {
			continue
		}
		isDisabled, found := dp.Attributes.Value(odigosattributes.SamplingRuleDisabled)
		require.True(t, found, "a disabled rule's metrics must be marked as disabled")
		assert.True(t, isDisabled.AsBool())
		disabledRuleMatches += dp.Value
	}
	assert.Equal(t, int64(1), disabledRuleMatches)
}
