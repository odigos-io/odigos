package odigossqldboperationprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestComponentFactoryType(t *testing.T) {
	require.Equal(t, "odigossqldboperationprocessor", NewFactory().Type().String())
}

func TestComponentConfigStruct(t *testing.T) {
	require.NoError(t, componenttest.CheckConfigStruct(NewFactory().CreateDefaultConfig()))
}

func TestComponentLifecycle(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name     string
		createFn func(ctx context.Context, set processor.Settings, cfg component.Config) (component.Component, error)
	}{

		{
			name: "traces",
			createFn: func(ctx context.Context, set processor.Settings, cfg component.Config) (component.Component, error) {
				return factory.CreateTracesProcessor(ctx, set, cfg, consumertest.NewNop())
			},
		},
	}

	cm, err := confmaptest.LoadConf("metadata.yaml")
	require.NoError(t, err)
	cfg := factory.CreateDefaultConfig()
	sub, err := cm.Sub("tests::config")
	require.NoError(t, err)
	require.NoError(t, sub.Unmarshal(&cfg))

	for _, test := range tests {
		t.Run(test.name+"-shutdown", func(t *testing.T) {
			c, err := test.createFn(context.Background(), processortest.NewNopSettings(), cfg)
			require.NoError(t, err)
			err = c.Shutdown(context.Background())
			require.NoError(t, err)
		})
		t.Run(test.name+"-lifecycle", func(t *testing.T) {
			c, err := test.createFn(context.Background(), processortest.NewNopSettings(), cfg)
			require.NoError(t, err)
			host := componenttest.NewNopHost()
			err = c.Start(context.Background(), host)
			require.NoError(t, err)
			require.NotPanics(t, func() {
				switch test.name {
				case "traces":
					e, ok := c.(processor.Traces)
					require.True(t, ok)
					traces := generateLifecycleTestTraces()
					if !e.Capabilities().MutatesData {
						traces.MarkReadOnly()
					}
					err = e.ConsumeTraces(context.Background(), traces)
				}
			})
			require.NoError(t, err)
			err = c.Shutdown(context.Background())
			require.NoError(t, err)
		})
	}
}

func generateLifecycleTestTraces() ptrace.Traces {
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("resource", "R1")
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.Attributes().PutStr("test_attr", "value_1")
	span.SetName("test_span")
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now().Add(-1 * time.Second)))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	return traces
}
