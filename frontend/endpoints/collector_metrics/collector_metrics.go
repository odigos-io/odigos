package collectormetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/common/consts"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const (
	/* Sustained rates of otelcol_exporter_send_failed_spans and otelcol_exporter_send_failed_metric_points 
	indicate that the Collector is not able to export data as expected.
	These metrics do not inherently imply data loss since there could be retries.
	But a high rate of failures could indicate issues with the network or backend receiving the data */
	exportFailSpansMetric = "otelcol_exporter_send_failed_spans"
	exportSuccessSpansMetric = "otelcol_exporter_sent_spans"
)

type odigosMetricsConsumer struct {
	totalSentSpans int
	totalFailedSpans int

	lastIntervalSentSpans int
	lastIntervalFailedSpans int

	lastIntervalTimeStamp time.Time
}

func (c *odigosMetricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func  (c *odigosMetricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	firstBatch := c.lastIntervalTimeStamp.IsZero()
	if firstBatch {
		c.lastIntervalTimeStamp = time.Now()
	}
	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		smSlice := rm.At(i).ScopeMetrics()
		res := rm.At(i).Resource()
		resMap := res.Attributes().AsRaw()
		fmt.Printf("Resource: %v\n", resMap)
		for j := 0; j < smSlice.Len(); j++ {
			sm := smSlice.At(j)
			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				fmt.Printf("metric metadata %v\n", m.Metadata().AsRaw())
				switch m.Name() {
					case exportFailSpansMetric:
						newTotalFailedSpans := int(m.Sum().DataPoints().At(0).DoubleValue())
						if !firstBatch {
							c.lastIntervalFailedSpans = newTotalFailedSpans - c.totalFailedSpans
						}
						c.totalFailedSpans = newTotalFailedSpans
					case exportSuccessSpansMetric:
						newTotalSentSpans := int(m.Sum().DataPoints().At(0).DoubleValue())
						if !firstBatch {
							c.lastIntervalSentSpans = newTotalSentSpans - c.totalSentSpans
						}
						c.totalSentSpans = newTotalSentSpans
				}
			}
		}
	}

	fmt.Printf("state after consuming metrics: %+v\n", c)
	return nil
}


func SetupOTLPReceiver(ctx context.Context) {
	f := otlpreceiver.NewFactory()

	cfg, ok := f.CreateDefaultConfig().(*otlpreceiver.Config)
	if !ok {
		panic("failed to cast default config to otlpreceiver.Config")
	}

	cfg.GRPC.NetAddr.Endpoint = fmt.Sprintf("0.0.0.0:%d", consts.OTLPPort)

	r, err := f.CreateMetricsReceiver(ctx, receivertest.NewNopSettings(), cfg, &odigosMetricsConsumer{})
	if err != nil {
		panic("failed to create receiver")
	}

	r.Start(ctx, componenttest.NewNopHost())
	defer r.Shutdown(ctx)

	fmt.Print("OTLP Receiver is running\n")
	<-ctx.Done()

}