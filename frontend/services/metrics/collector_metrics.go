package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const (
	// Accepted spans, from two different receivers depending on the collector role.
	// OTel built-in receiver self-telemetry (gateway pods / OTLP receiver).
	metricOtelReceiverAcceptedSpans = "otelcol_receiver_accepted_spans"
	// Odigos eBPF receiver (node collector / odiglet pods).
	metricEBPFReceiverAcceptedSpans = "otelcol_odigos_ebpf_accepted_spans"

	// Refused (we also call it dropped) spans, from two different receivers depending on the collector role.
	// Receiver drop/refuse counters from collector self-telemetry
	metricOtelReceiverRefusedSpans = "otelcol_receiver_refused_spans"
	// Samples lost while reading the eBPF buffer.
	metricEBPFReceiverLostSamples = "otelcol_odigos_ebpf_lost_samples"

	// Exporter success/failure counters from collector self-telemetry.
	metricExporterSentSpans       = "otelcol_exporter_sent_spans"
	metricExporterSendFailedSpans = "otelcol_exporter_send_failed_spans"

	// Python instrumentation metrics.
	metricPythonSentSpans     = "odigos_python_ebpf_instrumentation_sent_events"
	metricPythonFailedSpans   = "odigos_python_ebpf_instrumentation_output_failed_events"
	metricPythonTooLargeSpans = "odigos_python_ebpf_instrumentation_events_too_larger"

	// NodeJS instrumentation metrics
	metricNodeJSSentSpans     = "odigos_nodejs_ebpf_instrumentation_sent_events"
	metricNodeJSFailedSpans   = "odigos_nodejs_ebpf_instrumentation_output_failed_events"
	metricNodeJSTooLargeSpans = "odigos_nodejs_ebpf_instrumentation_events_too_larger"

	// eBPF core metrics
	metricEBPFCoreSentEvents   = "odigos_ebpf_events_sent"
	metricEBPFCoreFailedEvents = "odigos_ebpf_events_send_failed"
)

type PodRates struct {
	MetricsAcceptedRps float64
	// Spans lost/refused at the receiver itself (e.g. eBPF buffer overflow, memory limiter).
	MetricsDroppedRps  float64
	ExporterSuccessRps float64
	ExporterFailedRps  float64
	Window             string
	LastScrape         time.Time
}

// GetCollectorMetricsFromMetricsStore queries own-metrics (VictoriaMetrics via Prometheus-compatible API)
// and aggregates per-pod rates for accepted/dropped spans and exporter success/failures.
// Works for data collection and gateway pods.
func GetCollectorMetricsFromMetricsStore(ctx context.Context, api v1.API, namespace string, podNames []string, window string) (map[string]PodRates, error) {
	if api == nil {
		return nil, fmt.Errorf("own-metrics API is nil")
	}
	if window == "" {
		window = DefaultMetricsWindow
	}
	if namespace == "" {
		namespace = env.GetCurrentNamespace()
	}
	if len(podNames) == 0 {
		return map[string]PodRates{}, nil
	}

	podRegex := buildPodRegex(podNames)
	now := time.Now()

	// Metrics from the OTel receiver
	queryOtelReceiverAccepted := rateSumByPod(metricOtelReceiverAcceptedSpans, podRegex, window)
	queryOtelReceiverRefused := rateSumByPod(metricOtelReceiverRefusedSpans, podRegex, window)
	// Metrics from the OTel exporter
	queryExpSent := rateSumByPod(metricExporterSentSpans, podRegex, window)
	queryExpFailed := rateSumByPod(metricExporterSendFailedSpans, podRegex, window)
	// Metrics from the Odigos eBPF receiver
	queryEBPFReceiverAccepted := rateSumByPod(metricEBPFReceiverAcceptedSpans, podRegex, window)
	queryEBPFReceiverDropped := rateSumByPod(metricEBPFReceiverLostSamples, podRegex, window)
	// Python metrics
	queryPythonSent := rateSumByPod(metricPythonSentSpans, podRegex, window)
	queryPythonDropped := rateSumByPod(fmt.Sprintf("(%s|%s)", metricPythonFailedSpans, metricPythonTooLargeSpans), podRegex, window)
	// NodeJS metrics
	queryNodeJSSent := rateSumByPod(metricNodeJSSentSpans, podRegex, window)
	queryNodeJSDropped := rateSumByPod(fmt.Sprintf("(%s|%s)", metricNodeJSFailedSpans, metricNodeJSTooLargeSpans), podRegex, window)
	// eBPF core (go/c++ shared-buffer) metrics
	queryEBPFCoreSent := rateSumByPod(metricEBPFCoreSentEvents, podRegex, window)
	queryEBPFCoreFailed := rateSumByPod(metricEBPFCoreFailedEvents, podRegex, window)

	otelReceiverAccepted, tsAcc, err := queryVector(ctx, api, queryOtelReceiverAccepted, now)
	if err != nil {
		return nil, err
	}
	otelReceiverRefused, tsRef, err := queryVector(ctx, api, queryOtelReceiverRefused, now)
	if err != nil {
		return nil, err
	}
	expSent, tsSent, err := queryVector(ctx, api, queryExpSent, now)
	if err != nil {
		return nil, err
	}
	expFailed, tsFail, err := queryVector(ctx, api, queryExpFailed, now)
	if err != nil {
		return nil, err
	}
	ebpfReceiverAccepted, tsEBPFAccept, err := queryVector(ctx, api, queryEBPFReceiverAccepted, now)
	if err != nil {
		return nil, err
	}
	ebpfReceiverDropped, tsEBPFDrop, err := queryVector(ctx, api, queryEBPFReceiverDropped, now)
	if err != nil {
		return nil, err
	}
	pythonSent, tsPythonSent, err := queryVector(ctx, api, queryPythonSent, now)
	if err != nil {
		return nil, err
	}
	pythonDropped, tsPythonDropped, err := queryVector(ctx, api, queryPythonDropped, now)
	if err != nil {
		return nil, err
	}
	nodeJSSent, tsNodeJSSent, err := queryVector(ctx, api, queryNodeJSSent, now)
	if err != nil {
		return nil, err
	}
	nodeJSDropped, tsNodeJSDropped, err := queryVector(ctx, api, queryNodeJSDropped, now)
	if err != nil {
		return nil, err
	}
	ebpfCoreSent, tsEBPFCoreSent, err := queryVector(ctx, api, queryEBPFCoreSent, now)
	if err != nil {
		return nil, err
	}
	ebpfCoreFailed, tsEBPFCoreFailed, err := queryVector(ctx, api, queryEBPFCoreFailed, now)
	if err != nil {
		return nil, err
	}
	result := make(map[string]PodRates, len(podNames))
	for _, pod := range podNames {
		result[pod] = PodRates{
			Window: window,
		}
	}

	lastScrape := maxTime(tsAcc, tsRef, tsSent, tsFail, tsEBPFAccept, tsEBPFDrop, tsPythonSent, tsPythonDropped, tsNodeJSSent, tsNodeJSDropped, tsEBPFCoreSent, tsEBPFCoreFailed)

	for pod := range result {
		r := result[pod]

		// A single pod can receive spans through both the eBPF receiver and the OTel built-in receiver, since not all instrumentations go through eBPF.
		// Sum both receiver sources for accepted.
		// refused/dropped spans are aggregated under dropped.
		r.MetricsAcceptedRps = sumByPod(pod, otelReceiverAccepted, ebpfReceiverAccepted, pythonSent, nodeJSSent, ebpfCoreSent)
		r.MetricsDroppedRps = sumByPod(pod, ebpfReceiverDropped, otelReceiverRefused, pythonDropped, nodeJSDropped, ebpfCoreFailed)

		if v, ok := expSent[pod]; ok {
			r.ExporterSuccessRps = v
		}
		if v, ok := expFailed[pod]; ok {
			r.ExporterFailedRps = v
		}
		r.LastScrape = lastScrape
		result[pod] = r
	}

	return result, nil
}
