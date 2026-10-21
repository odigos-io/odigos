package odigostailsamplingprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/internal/metadatatest"
	commonapi "github.com/odigos-io/odigos/common/api"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	odigoscollector "github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

const testWorkloadKey = "default/Deployment/frontend/app"

// fakeConfigExtension is a minimal OdigosConfigExtension that serves one workload config
// for every resource, so the processor can be driven without a Kubernetes cluster.
type fakeConfigExtension struct {
	component.StartFunc
	component.ShutdownFunc
	cfg *commonapi.ContainerCollectorConfig
}

func (f *fakeConfigExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return f.cfg, true
}

func (f *fakeConfigExtension) IsActiveSource(pcommon.Resource) bool { return true }

func (f *fakeConfigExtension) GetWorkloadCacheKey(pcommon.Resource) (string, error) {
	return testWorkloadKey, nil
}

func (f *fakeConfigExtension) GetWorkloadIdentityFromResource(pcommon.Resource) (string, pcommon.Map, error) {
	return testWorkloadKey, pcommon.NewMap(), nil
}

func (f *fakeConfigExtension) RegisterWorkloadConfigCacheCallback(cb odigoscollector.WorkloadConfigCacheCallback) {
	cb.OnSet(testWorkloadKey, f.cfg)
}

func (f *fakeConfigExtension) UnregisterWorkloadConfigCacheCallback(odigoscollector.WorkloadConfigCacheCallback) {
}

func (f *fakeConfigExtension) WaitForCacheSync(context.Context) bool { return true }

func (f *fakeConfigExtension) GetDataStreamsForWorkload(pcommon.Resource) ([]string, bool) {
	return nil, false
}

type fakeHost struct {
	extensions map[component.ID]component.Component
}

func (h fakeHost) GetExtensions() map[component.ID]component.Component { return h.extensions }

// startProcessorWithCostReductionRule wires the real processor to a fake config extension holding
// a single cost-reduction rule that matches every span and samples at the given percentage.
func startProcessorWithCostReductionRule(t *testing.T, tt *componenttest.Telemetry, percentageAtMost float64) processor.Traces {
	t.Helper()

	extID := component.MustNewID("odigosconfigk8s")
	cfg := &Config{OdigosConfigExtension: &extID}

	proc, err := NewFactory().CreateTraces(context.Background(), metadatatest.NewSettings(tt), cfg, consumertest.NewNop())
	require.NoError(t, err)

	ext := &fakeConfigExtension{
		cfg: &commonapi.ContainerCollectorConfig{
			ContainerName: "app",
			TailSampling: &commonapisampling.TailSamplingSourceConfig{
				CostReductionRules: []commonapisampling.CostReductionRule{
					{
						Id:               "rule-1",
						Name:             "match everything",
						PercentageAtMost: percentageAtMost,
					},
				},
			},
		},
	}

	host := fakeHost{extensions: map[component.ID]component.Component{extID: ext}}
	require.NoError(t, proc.Start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, proc.Shutdown(context.Background())) })

	return proc
}

// sumDataPoints returns the summed value of the data points whose attribute set equals want.
func sumDataPoints(t *testing.T, tt *componenttest.Telemetry, metricName string, want attribute.Set) int64 {
	t.Helper()

	got, err := tt.GetMetric(metricName)
	require.NoError(t, err)

	sum, ok := got.Data.(metricdata.Sum[int64])
	require.True(t, ok, "metric %s is not an int64 sum", metricName)

	var total int64
	for _, dp := range sum.DataPoints {
		if dp.Attributes.Equals(&want) {
			total += dp.Value
		}
	}
	return total
}

// The span-level sampling counters must report the number of spans in the trace, not the
// number of traces. The unattributed total and the per-rule series are derived from two
// different code paths and have to agree.
func TestProcessorRecordsEverySpanInTheSamplingCounters(t *testing.T) {
	const spansInTrace = 9

	costReductionAttrs := attribute.NewSet(
		attribute.String(odigosattributes.SamplingCategory, string(consts.SamplingCategoryCostReduction)),
	)
	ruleAttrs := attribute.NewSet(
		attribute.String(odigosattributes.SamplingCategory, string(consts.SamplingCategoryCostReduction)),
		attribute.String(odigosattributes.SamplingRuleId, "rule-1"),
		attribute.String(odigosattributes.SamplingRuleName, "match everything"),
	)

	tests := []struct {
		name             string
		percentageAtMost float64
		decisionMetric   string
	}{
		{
			name:             "trace kept",
			percentageAtMost: 100,
			decisionMetric:   "otelcol_odigos.sampling.span.keep_count",
		},
		{
			name:             "trace dropped",
			percentageAtMost: 0,
			decisionMetric:   "otelcol_odigos.sampling.span.drop_count",
		},
	}

	for _, tt2 := range tests {
		t.Run(tt2.name, func(t *testing.T) {
			tt := componenttest.NewTelemetry()
			t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

			proc := startProcessorWithCostReductionRule(t, tt, tt2.percentageAtMost)

			td := ptrace.NewTraces()
			traceID := pcommon.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
			appendSpans(td, traceID, spansInTrace)

			require.NoError(t, proc.ConsumeTraces(context.Background(), td))

			assert.Equal(t, int64(spansInTrace),
				sumDataPoints(t, tt, "otelcol_odigos.sampling.span.check_count", *attribute.EmptySet()),
				"total spans checked")
			assert.Equal(t, int64(spansInTrace),
				sumDataPoints(t, tt, "otelcol_odigos.sampling.span.check_count", ruleAttrs),
				"spans checked by the cost reduction rule")
			assert.Equal(t, int64(spansInTrace),
				sumDataPoints(t, tt, tt2.decisionMetric, costReductionAttrs),
				"spans in the cost reduction sampling decision")

			assert.Equal(t, int64(1),
				sumDataPoints(t, tt, "otelcol_odigos.sampling.trace.check_count", *attribute.EmptySet()),
				"total traces checked")
		})
	}
}
