package metrics

import (
	"context"

	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	otelMeterName   = "github.com/odigos.io/odigos/odiglet"
	languageAttrKey = "language"
	typeAttrKey     = "type"
)

var meter = otel.Meter(otelMeterName)

// OdigletMetrics tracks unified metrics for all instrumented pods
type OdigletMetrics struct {
	ebpfManager     instrumentation.Manager
	connectionCache *connection.ConnectionsCache

	// instrumentedPodsByLanguage tracks all instrumented pods (both eBPF and native) per language
	instrumentedPodsByLanguage otelmetric.Int64ObservableGauge

	registration otelmetric.Registration
}

// NewOdigletMetrics creates unified metrics that aggregate from both eBPF manager and OpAMP connections
func NewOdigletMetrics(ebpfManager instrumentation.Manager, connectionCache *connection.ConnectionsCache) (*OdigletMetrics, error) {
	m := &OdigletMetrics{
		ebpfManager:     ebpfManager,
		connectionCache: connectionCache,
	}

	var err error
	m.instrumentedPodsByLanguage, err = meter.Int64ObservableGauge(
		"odigos.odiglet.instrumented_pods_by_language",
		otelmetric.WithDescription("Number of instrumented pods per programming language"),
		otelmetric.WithUnit("{pod}"),
	)
	if err != nil {
		return nil, err
	}

	// Register callback to observe the gauge
	m.registration, err = meter.RegisterCallback(
		m.observeInstrumentedPods,
		m.instrumentedPodsByLanguage,
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// observeInstrumentedPods is the callback that observes instrumented pod counts by language
func (m *OdigletMetrics) observeInstrumentedPods(ctx context.Context, observer otelmetric.Observer) error {
	// Get eBPF instrumented process counts
	if m.ebpfManager != nil {
		ebpfCounts := m.ebpfManager.GetInstrumentedCountsByLanguage()
		for language, count := range ebpfCounts {
			observer.ObserveInt64(m.instrumentedPodsByLanguage, int64(count),
				otelmetric.WithAttributes(
					attribute.String(languageAttrKey, language),
					attribute.String(typeAttrKey, "ebpf"),
				))
		}
	}

	// Get native agent connection counts
	if m.connectionCache != nil {
		nativeCounts := m.connectionCache.GetConnectionCountsByLanguage()
		for language, count := range nativeCounts {
			observer.ObserveInt64(m.instrumentedPodsByLanguage, int64(count),
				otelmetric.WithAttributes(
					attribute.String(languageAttrKey, language),
					attribute.String(typeAttrKey, "native"),
				))
		}
	}

	return nil
}

// Close unregisters the metrics callback
func (m *OdigletMetrics) Close() error {
	if m.registration != nil {
		return m.registration.Unregister()
	}
	return nil
}
