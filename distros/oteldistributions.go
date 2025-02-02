package distros

import "github.com/odigos-io/odigos/common"

type ImplementationType string

const (
	// Native is for implementation in the language of the application and
	// is integrated into the application code via runtime support (e.g. Java agent).
	NativeImplementation ImplementationType = "native"

	// EbpfImplementation implements using eBPF code injected into the application process.
	EbpfImplementation ImplementationType = "ebpf"

	// TODO: in the future we can specify more implementation code types like: "integrated".
)

type OtelInstrumentationsMetadata struct {
	// The implementation type of the instrumentations
	Implementation ImplementationType `json:"implementation"`
}

type OtelSdkMetadata struct {
	// The implementation type of the SDK
	Implementation ImplementationType `json:"implementation"`
}

type RuntimeEnvironment struct {
	// the runtime environment this distribution targets.
	// examples: nodejs, JVM, CPython, etc.
	// while java-script can run in both nodejs and browser, the distribution should specify where it is intended to run.
	RuntimeEnvironmentName string `json:"runtimeEnvironmentName"`

	// semconv range of the runtime versions supported by this distribution.
	RuntimeEnvironmentVersion string `json:"runtimeEnvironmentVersion"`
}

type Framwork struct {
	// the framework this distribution targets.
	FrameworkName string `json:"frameworkName"`

	// semconv range of the framework versions supported by this distribution.
	FrameworkVersion string `json:"frameworkVersion"`
}

// OtelDistro (Short for OpenTelemetry Distribution) is a collection of OpenTelemetry components,
// including instrumentations, SDKs, and other components that are distributed together.
// Each distribution includes a unique name, and metadata about the ways it is implemented.
// The metadata includes the tiers of the distribution, the instrumentations, and the SDKs used.
// Multiple distributions can co-exist with the same properties but different names.
type OtelDistro struct {

	// a unique name for this distribution, which helps to identify it.
	// should be a single word, lowercase, and may include hyphens (nodejs-community, dotnet-legacy-instrumentation).
	Name string `json:"name"`

	// the programming language this distribution targets.
	// each distribution must target a single language.
	Language common.ProgrammingLanguage `json:"language"`

	// the runtime environments this distribution targets.
	// examples: nodejs, JVM, CPython, etc.
	// while java-script can run in both nodejs and browser, the distribution should specify where it is intended to run.
	RuntimeEnvironments []RuntimeEnvironment `json:"runtimeEnvironments"`

	// A list of frameworks this distribution targets (can be left empty)
	Frameworks []Framwork `json:"frameworks"`

	// a human-friendly name for this distribution, which can be displayed in the UI and documentation.
	// may include spaces and special characters.
	DisplayName string `json:"displayName"`

	// Free text description of the distribution, what it includes, it's use cases, etc.
	Description string `json:"description"`

	// Specifies the odigos tiers which includes this distribution
	Tiers []common.OdigosTier `json:"tiers"`

	// describe how the instrumentations are implemented in the distribution
	// instrumentations is the code that calls "startSpan" and runs for each operation being recorded.
	Instrumentations OtelInstrumentationsMetadata `json:"instrumentations"`

	// describe the OpenTelemetry SDKs used in the distribution
	// SDK is the code that process and exports a span recorded by instrumentation.
	OtelSdk OtelSdkMetadata `json:"otelSdk"`
}
