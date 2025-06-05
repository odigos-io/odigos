package instrumentation

import (
	"errors"

	otelmetric "go.opentelemetry.io/otel/metric"
)

type managerMetrics struct {
	instrumentedProcesses otelmetric.Int64UpDownCounter

	failedInstrumentations otelmetric.Int64Counter
}

func newManagerMetrics(meter otelmetric.Meter) (*managerMetrics, error) {
	var err, errs error
	m := &managerMetrics{}

	m.instrumentedProcesses, err = meter.Int64UpDownCounter(
		"odigos.ebpf.instrumentation.manager.instrumented_processes",
		otelmetric.WithDescription("Number of processes currently instrumented"),
		otelmetric.WithUnit("{process}"),
	)
	errs = errors.Join(errs, err)

	m.failedInstrumentations, err = meter.Int64Counter(
		"odigos.ebpf.instrumentation.manager.failed_instrumentations",
		otelmetric.WithDescription("Number of processes that failed to be instrumented"),
		otelmetric.WithUnit("{process}"),
	)
	errs = errors.Join(errs, err)

	return m, errs
}
