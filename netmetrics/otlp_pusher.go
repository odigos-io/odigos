package netmetrics

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// OTLPPusher periodically scrapes OBI's raw flows, resolves them to service identity,
// and pushes them to a local OTLP/gRPC collector as the cumulative counter
// network.flow.bytes (OTel-semconv attributes). This is the production export path:
// the enriched metrics ride the agent's existing collector pipeline to any destination,
// instead of being scraped from a Prometheus endpoint. Shared by vm-agent and odiglet.
type OTLPPusher struct {
	exporter  *otlpmetricgrpc.Exporter
	enricher  *PrometheusEnricher
	resolver  *ServiceResolver
	interval  time.Duration
	scopeName string
}

// NewOTLPPusher dials endpoint (host:port, insecure — local collector) and prepares a
// pusher that resolves flows via resolver and scrapes raw flows from obiURL.
func NewOTLPPusher(ctx context.Context, endpoint, obiURL string, resolver *ServiceResolver, interval time.Duration) (*OTLPPusher, error) {
	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}
	return &OTLPPusher{
		exporter:  exp,
		enricher:  NewPrometheusEnricher(obiURL, resolver),
		resolver:  resolver,
		interval:  interval,
		scopeName: "github.com/odigos-io/odigos/netmetrics",
	}, nil
}

// Run pushes enriched metrics on each interval until ctx is cancelled, then flushes
// and shuts the exporter down.
func (p *OTLPPusher) Run(ctx context.Context) {
	start := time.Now()
	t := time.NewTicker(p.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = p.exporter.Shutdown(shutCtx)
			cancel()
			return
		case <-t.C:
			rm, err := p.build(start, time.Now())
			if err != nil {
				slog.Warn("netmetrics: scrape OBI for OTLP push failed", "err", err)
				continue
			}
			pushCtx, cancel := context.WithTimeout(ctx, p.interval)
			if err := p.exporter.Export(pushCtx, rm); err != nil {
				slog.Warn("netmetrics: OTLP export failed", "err", err)
			}
			cancel()
		}
	}
}

// build resolves the current flows and renders them as one cumulative Sum metric.
func (p *OTLPPusher) build(start, now time.Time) (*metricdata.ResourceMetrics, error) {
	flows, err := p.enricher.ResolveFlows()
	if err != nil {
		return nil, err
	}
	dps := make([]metricdata.DataPoint[int64], 0, len(flows))
	for _, f := range flows {
		dps = append(dps, metricdata.DataPoint[int64]{
			Attributes: attribute.NewSet(
				attribute.String("service.name", f.Service),
				attribute.String("peer.service.name", f.Peer),
				attribute.String("network.transport", f.Transport),
				attribute.String("network.io.direction", f.Direction),
				attribute.String("server.port", f.ServerPort),
			),
			StartTime: start,
			Time:      now,
			Value:     int64(f.Bytes),
		})
	}
	return &metricdata.ResourceMetrics{
		Resource: resource.NewSchemaless(semconv.ServiceName("odigos-netmetrics")),
		ScopeMetrics: []metricdata.ScopeMetrics{{
			Scope: instrumentation.Scope{Name: p.scopeName},
			Metrics: []metricdata.Metrics{{
				Name:        "network.flow.bytes",
				Description: "Bytes between services (OBI flow enriched with service identity)",
				Unit:        "By",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints:  dps,
				},
			}},
		}},
	}, nil
}
