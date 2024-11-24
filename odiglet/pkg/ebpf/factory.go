package ebpf

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"

	"go.opentelemetry.io/otel/attribute"
)

type Settings struct {
	ServiceName        string
	ResourceAttributes []attribute.KeyValue
	InitialConfig	   *odigosv1.SdkConfig
}

// Factory is used to create an Instrumentation
type Factory interface {
	// CreateInstrumentation will initialize the instrumentation for the given process.
	// Setting can be used to pass initial configuration to the instrumentation.
	CreateInstrumentation(ctx context.Context, pid int, settings Settings) (Instrumentation, error)
}

type FactoryID struct {
	Language common.ProgrammingLanguage
	OtelSdk  common.OtelSdk
}

// Instrumentation is used to instrument a running process
type Instrumentation interface {
	// Loads the relevant probes, and will perform any initialization required
	// for the instrumentation to be ready to run.
	// For eBPF, this will load the probes into the kernel
	// In case of a failure, an error will be returned and all the resources will be cleaned up.
	Load(ctx context.Context) error

	// Run will attach the probes to the relevant process, and will start the instrumentation.
	// It is a blocking call, and will return only when the instrumentation is stopped.
	// During the run, telemetry will be collected from the probes and sent with the configured exporter.
	// Run will return when either a fatal error occurs, the context is canceled, or Close is called.
	Run(ctx context.Context) error

	// Close will stop the instrumentation (Stop the Run function) and clean up all the resources associated with it.
	// When it returns, the instrumentation is stopped and all resources are cleaned up.
	Close(ctx context.Context) error

	// ApplyConfig will send a configuration update to the instrumentation.
	ApplyConfig(ctx context.Context, config *odigosv1.SdkConfig) error
}
