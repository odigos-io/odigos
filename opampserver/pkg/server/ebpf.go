package server

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
)

type EbpfHooks interface {

	// This function is called when a new opamp client connection is established.
	// If set, it is expected for this callback to load any eBPF programs needed for instrumentation
	// of this client.
	// The function should block until the eBPF programs are loaded, either with success or error.
	// If an error is returned, the agent will be signaled not to start the and connection will be closed.
	// If the function returns nil, the connection will be allowed to proceed, and the eBPF part is assumed ready.
	//
	// Input:
	// - ctx: the context of the request
	// - programmingLanguage: the programming language of the agent, as reported by the agent, conforming to otel semconv
	// - pid: the process id of the agent process, which is used to inject the eBPF programs
	// - serviceName: the service name to use as resource attribute for generated telemetry
	// - resourceAttributes: a list of resource attributes to populate in the resource of the generated telemetry
	//
	// Output:
	// - error: if an error occurred during the loading of the eBPF programs
	// - cancelFunc: if loaded successfully, a cancel function to be called when the connection is closed to unload the eBPF programs and release resources
	// at the moment, errors from cancel are logged and ignored
	OnNewInstrumentedProcess(ctx context.Context, programmingLanguage string, pid int64, serviceName string, resourceAttributes []attribute.KeyValue) (func() error, error)
}
