package instrumentation

import (
	"errors"

	otelmetric "go.opentelemetry.io/otel/metric"
)

type managerMetrics struct {
	instrumentedProcesses otelmetric.Int64UpDownCounter

	failedInstrumentations otelmetric.Int64Counter
	
	// New eBPF-specific metrics
	ebpfLoadedPrograms      otelmetric.Int64UpDownCounter
	ebpfLoadedMaps          otelmetric.Int64UpDownCounter
	ebpfLoadedLinks         otelmetric.Int64UpDownCounter
	ebpfProgramLoadDuration otelmetric.Int64Histogram
	ebpfMapLookups          otelmetric.Int64Counter
	ebpfMapUpdates          otelmetric.Int64Counter
	ebpfProgramErrors       otelmetric.Int64Counter
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

	m.ebpfLoadedPrograms, err = meter.Int64UpDownCounter(
		"odigos.ebpf.instrumentation.manager.loaded_programs",
		otelmetric.WithDescription("Number of eBPF programs currently loaded by the instrumentation manager"),
		otelmetric.WithUnit("{program}"),
	)
	errs = errors.Join(errs, err)

	m.ebpfLoadedMaps, err = meter.Int64UpDownCounter(
		"odigos.ebpf.instrumentation.manager.loaded_maps",
		otelmetric.WithDescription("Number of eBPF maps currently loaded by the instrumentation manager"),
		otelmetric.WithUnit("{map}"),
	)
	errs = errors.Join(errs, err)

	m.ebpfLoadedLinks, err = meter.Int64UpDownCounter(
		"odigos.ebpf.instrumentation.manager.loaded_links",
		otelmetric.WithDescription("Number of eBPF links currently loaded by the instrumentation manager"),
		otelmetric.WithUnit("{link}"),
	)
	errs = errors.Join(errs, err)

	m.ebpfProgramLoadDuration, err = meter.Int64Histogram(
		"odigos.ebpf.instrumentation.manager.program_load_duration_ms",
		otelmetric.WithDescription("Time taken to load eBPF programs"),
		otelmetric.WithUnit("ms"),
	)
	errs = errors.Join(errs, err)

	m.ebpfMapLookups, err = meter.Int64Counter(
		"odigos.ebpf.instrumentation.manager.map_lookups_total",
		otelmetric.WithDescription("Total number of eBPF map lookups performed"),
		otelmetric.WithUnit("{lookup}"),
	)
	errs = errors.Join(errs, err)

	m.ebpfMapUpdates, err = meter.Int64Counter(
		"odigos.ebpf.instrumentation.manager.map_updates_total",
		otelmetric.WithDescription("Total number of eBPF map updates performed"),
		otelmetric.WithUnit("{update}"),
	)
	errs = errors.Join(errs, err)

	m.ebpfProgramErrors, err = meter.Int64Counter(
		"odigos.ebpf.instrumentation.manager.program_errors_total",
		otelmetric.WithDescription("Total number of eBPF program execution errors"),
		otelmetric.WithUnit("{error}"),
	)
	errs = errors.Join(errs, err)

	return m, errs
}
