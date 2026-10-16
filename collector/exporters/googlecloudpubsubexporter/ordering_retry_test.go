// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googlecloudpubsubexporter

import (
	"context"
	"errors"
	"testing"

	pb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type failFirstPublisher struct {
	failed   bool
	messages []*pb.PubsubMessage
}

func (p *failFirstPublisher) Close() error { return nil }
func (p *failFirstPublisher) Publish(_ context.Context, request *pb.PublishRequest, _ ...gax.CallOption) (*pb.PublishResponse, error) {
	if !p.failed {
		p.failed = true
		return nil, errors.New("transient publish failure")
	}
	p.messages = append(p.messages, request.Messages...)
	return &pb.PublishResponse{}, nil
}

func TestOrderedPublishRetryPreservesResources(t *testing.T) {
	for _, signal := range []string{"traces", "metrics", "logs"} {
		t.Run(signal, func(t *testing.T) {
			ex, _ := newTestExporter(t, func(cfg *Config) { cfg.Ordering.Enabled = true; cfg.Ordering.FromResourceAttribute = "service.name" })
			publisher := &failFirstPublisher{}
			ex.client = publisher
			traces := ptrace.NewTraces()
			metrics := pmetric.NewMetrics()
			logs := plog.NewLogs()
			for _, name := range []string{"frontend", "backend"} {
				rs := traces.ResourceSpans().AppendEmpty()
				rs.Resource().Attributes().PutStr("service.name", name)
				rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName(name)
				rm := metrics.ResourceMetrics().AppendEmpty()
				rm.Resource().Attributes().PutStr("service.name", name)
				m := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
				m.SetName(name)
				m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)
				rl := logs.ResourceLogs().AppendEmpty()
				rl.Resource().Attributes().PutStr("service.name", name)
				rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(name)
			}
			consume := func() error {
				switch signal {
				case "traces":
					return ex.consumeTraces(t.Context(), traces)
				case "metrics":
					return ex.consumeMetrics(t.Context(), metrics)
				default:
					return ex.consumeLogs(t.Context(), logs)
				}
			}
			require.Error(t, consume())
			publisher.messages = nil
			require.NoError(t, consume())
			require.Len(t, publisher.messages, 2, "retry must republish both resources")
		})
	}
}
