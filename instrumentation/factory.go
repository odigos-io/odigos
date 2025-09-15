package instrumentation

import (
	"context"

	"github.com/cilium/ebpf"
	"go.opentelemetry.io/otel/attribute"
)

type Config any

// Settings is used to pass initial configuration to the instrumentation
type Settings struct {
	// ServiceName is the name of the service that is being instrumented
	// It will be used to populate the service.name resource attribute.
	ServiceName string
	// ResourceAttributes can be used to pass additional resource attributes to the instrumentation
	// These attributes will be added to the resource attributes of the telemetry data.
	ResourceAttributes []attribute.KeyValue
	// InitialConfig is the initial configuration that should be applied to the instrumentation,
	// it can be used to enable/disable specific instrumentation libraries, configure sampling, etc.
	InitialConfig Config
	// TracesMap is the optional common eBPF map that will be used to send events from eBPF probes.
	TracesMap ReaderMap
}

type ReaderMap struct {
	// Map is the optional eBPF map that will be used to send events from eBPF probes.
	// It should be either a perf event array or a ring buffer.
	// This map can be common to multiple instrumentations across different processes to save resources.
	Map *ebpf.Map

	// ExternalReader indicates if the consumer from this map is external -
	// meaning the instrumentation does not read from this map directly,
	// but rather it is done by an external reader (e.g., a telemetry collector).
	ExternalReader bool
}

// Factory is used to create an Instrumentation
type Factory interface {
	// CreateInstrumentation will initialize the instrumentation for the given process.
	// Setting can be used to pass initial configuration to the instrumentation.
	CreateInstrumentation(ctx context.Context, pid int, settings Settings) (Instrumentation, error)
}

// Instrumentation is used to instrument a running process
type Instrumentation interface {
	// Loads the relevant probes, and will perform any initialization required
	// for the instrumentation to be ready to run.
	// For eBPF, this will load the probes into the kernel
	// In case of a failure, an error will be returned and all the resources will be cleaned up.
	Load(ctx context.Context) error

	// Run will start reading events from the probes and export them.
	// It is a blocking call, and will return only when the instrumentation is stopped.
	// During the run, telemetry will be collected from the probes and sent with the configured exporter.
	// Run will return when either a fatal error occurs, the context is canceled, or Close is called.
	Run(ctx context.Context) error

	// Close will stop the instrumentation (Stop the Run function) and clean up all the resources associated with it.
	// When it returns, the instrumentation is stopped and all resources are cleaned up.
	Close(ctx context.Context) error

	// ApplyConfig will send a configuration update to the instrumentation.
	ApplyConfig(ctx context.Context, config Config) error
}
