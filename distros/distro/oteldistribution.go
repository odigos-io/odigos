package distro

import (
	"text/template"

	"github.com/odigos-io/odigos/common"
)

const AgentPlaceholderDirectory = "{{ODIGOS_AGENTS_DIR}}"

const RuntimeVersionMajorMinorDistroParameterName = "RUNTIME_VERSION_MAJOR_MINOR"

type RuntimeEnvironment struct {
	// the runtime environment this distribution targets.
	// examples: nodejs, JVM, CPython, etc.
	// while java-script can run in both nodejs and browser, the distribution should specify where it is intended to run.
	Name string `yaml:"name"`

	// semconv range of the runtime versions supported by this distribution.
	SupportedVersions string `yaml:"supportedVersions,omitempty"`
}

type Framework struct {
	// the framework this distribution targets.
	FrameworkName string `yaml:"frameworkName"`

	// semconv range of the framework versions supported by this distribution.
	FrameworkVersion string `yaml:"frameworkVersion"`
}

// static name-value environment variable that need to be set in the application runtime.
type StaticEnvironmentVariable struct {
	// The name of the environment variable to set.
	EnvName string `yaml:"envName"`

	// The value of the environment variable to set.
	EnvValue string `yaml:"envValue"`

	// pre-parsed template that is ready to be executed with the relevant parameters.
	// if the EnvValue field is templated (e.g. contains {{.PARAM_NAME}}), this field is set.
	// it indicates that the value should be templated with the relevant parameters.
	// this field is calculated based on the EnvValue field, and is not pulled from the yaml.
	Template *template.Template
}

type EnvironmentVariables struct {

	// if this distribution runs an opamp client, add the environment variables that configures the server endpoint (local node)
	OpAmpClientEnvironments bool `yaml:"opAmpClientEnvironments,omitempty"`

	// set to true if this distribution uses the OTLP HTTP protocol to emit telemetry data to node collector.
	// if `true` the OTEL_EXPORTER_OTLP_ENDPOINT environment variable will be set to LocalTrafficOTLPHttpDataCollectionEndpoint
	OtlpHttpLocalNode bool `yaml:"otlpHttpLocalNode,omitempty"`

	// some exporters will error un-nicely or even crash a pod if they try to export
	// a signal with no receiver.
	// for distros that can dynamically get the enabled signals list, this is not an issue,
	// but distros that do not support it use env vars at time of pod creation to set OTEL
	// variables to enabled signals (and avoid collecting and exporting disabled signals).
	// notice that this value will be set once on node creation, and will not be updated,
	// thus it is recommended to use dynamic signal list if possible.
	SignalsAsStaticOtelEnvVars bool `yaml:"signalsAsStaticOtelEnvVars,omitempty"`

	// list of static environment variables that need to be set in the application runtime.
	StaticVariables []StaticEnvironmentVariable `yaml:"staticVariables,omitempty"`

	// list of environment variables that needs to be appended based on the existing value.
	AppendOdigosVariables []AppendOdigosEnvironmentVariable `yaml:"appendOdigosVariables,omitempty"`
}

// this struct describes environment variables that needs to be set in the application runtime
// to enable the distribution.
// This environment variable should be patched with odigos value if it already exists in the manifest.
// If this value is determined to arrive from Container Runtime during runtime inspection, this value should be set in manifest so not to break the application.
type AppendOdigosEnvironmentVariable struct {

	// The name of the environment variable to set or patch.
	EnvName string `yaml:"envName"`

	// A pattern to replace the existing value in case it exists.
	// An example is `{{ORIGINAL_ENV_VALUE}} -javaagent:{{ODIGOS_AGENTS_DIR}}/java/javaagent.jar`
	// while updating an env var value, the user can use pre-defined templates to interact with dynamic values:
	// - `{{ORIGINAL_ENV_VALUE}}` - the original value of the environment variable
	// - `{{ODIGOS_AGENTS_DIR}}` - where odigos directory is mounted in the container ('/var/odigos' for k8s, and other values for other platforms)
	//
	// This allows distros to update the value with control over:
	// - the delimiter used for this language
	// - if odigos value is prepended or appended (or both)
	// - reference the agents directory in a platform-agnostic way
	ReplacePattern string `yaml:"replacePattern"`
}

type RuntimeAgent struct {
	// The name of a directory where odigos agent files can be found.
	// The special value {{ODIGOS_AGENTS_DIR}} is replaced with the actual value at this platform.
	// K8s will mount this directory from the node fs to the container. other platforms may have different ways to make the directory accessible.
	DirectoryNames []string `yaml:"directoryNames"`

	// This field indicates that the agent populates k8s resource attributes via environment variables.
	// It targets distros odigos uses without wrapper or customization.
	// For eBPF, the resource attributes are set in code.
	// For opamp distros, the resource attributes are set in the opamp server.
	// We will eventually remove this field once all distros upgrade to dynamic resource attributes.
	K8sAttrsViaEnvVars bool `yaml:"k8sAttrsViaEnvVars,omitempty"`

	// If mounting of agent directory is achieved via k8s virtual device,
	// this field specifies the name of the device to inject into the resources part of the pods container spec.
	Device *string `yaml:"device,omitempty"`

	// Some of the agents might require a specific file to loaded before we can start the instrumentation.
	// This list contains the full path of the files that need to be opened for the agent to properly start.
	// All these paths must be contained in one of the directoryNames.
	FileOpenTriggers []string `yaml:"fileOpenTriggers,omitempty"`

	// If true, the agent supports ld-preload injection of "append" environment variables.
	LdPreloadInjectionSupported bool `yaml:"ldPreloadInjectionSupported,omitempty"`

	// If true, the agent supports wasp
	WaspSupported bool `yaml:"waspSupported,omitempty"`

	// If true, the instrumentation applied by this agent does not require application restart.
	NoRestartRequired bool `yaml:"noRestartRequired,omitempty"`
}

type SpanMetrics struct {
	// if true, the agent supports span metrics.
	Supported bool `yaml:"supported,omitempty"`
}

// configuration for this distro's support for metrics generated from the runtime agent.
type AgentMetrics struct {

	// configuration for this distro's support for agent span metrics.
	// these are span metrics that are generated directly in the agent,
	// unlike span metrics calculated at collectors which miss
	// head unsampled spans and spans dropped before reaching the collector.
	SpanMetrics *SpanMetrics `yaml:"spanMetrics,omitempty"`
}

type HeadSampling struct {
	// if true, the distro supports head sampling for health checks.
	Supported bool `yaml:"supported,omitempty"`

	// the attribute to check for head sampling url.path
	// support for old semantic convention ("http.target" -> "url.path")
	UrlPathAttributeKey string `yaml:"urlPathAttributeKey,omitempty"`

	// the attribute to check for head sampling http.method
	// support for old semantic convention ("http.method" -> "http.request.method")
	HttpRequestMethodAttributeKey string `yaml:"httpRequestMethodAttributeKey,omitempty"`
}

type HeadersCollection struct {
	// if true, the distro supports headers collection for health checks.
	Supported bool `yaml:"supported,omitempty"`
}

type UrlTemplatization struct {
	// if true, the distro supports applying URL templatization rules to traces in the agent.
	// useful when spanmetrics are calculated in the agent itself, and for head sampling to use correct route.
	Supported bool `yaml:"supported,omitempty"`
}

type Traces struct {
	// if set, the distro supports head sampling based on root spans of traces.
	HeadSampling *HeadSampling `yaml:"headSampling,omitempty"`

	// if set, the distro supports headers collection for http headers.
	HeadersCollection *HeadersCollection `yaml:"headersCollection,omitempty"`

	// if set, the distro supports applying URL templatization rules to traces in the agent.
	// useful when spanmetrics are calculated in the agent itself, and for head sampling to use correct route.
	UrlTemplatization *UrlTemplatization `yaml:"urlTemplatization,omitempty"`
}

// OtelDistro (Short for OpenTelemetry Distribution) is a collection of OpenTelemetry components,
// including instrumentations, SDKs, and other components that are distributed together.
// Each distribution includes a unique name, and metadata about the ways it is implemented.
// The metadata includes the tiers of the distribution, the instrumentations, and the SDKs used.
// Multiple distributions can co-exist with the same properties but different names.
type OtelDistro struct {

	// a human-friendly name for this distribution, which can be displayed in the UI and documentation.
	// may include spaces and special characters.
	DisplayName string `yaml:"displayName"`

	// a unique name for this distribution, which helps to identify it.
	// should be a single word, lowercase, and may include hyphens (nodejs-community, dotnet-legacy-instrumentation).
	Name string `yaml:"name"`

	// the programming language this distribution targets.
	// each distribution must target a single language.
	Language common.ProgrammingLanguage `yaml:"language"`

	// List of distribution parameters that are required to be set by the user.
	// for example: libc type.
	RequireParameters []string `yaml:"requireParameters,omitempty"`

	// the runtime environments this distribution targets.
	// examples: nodejs, JVM, CPython, etc.
	// while java-script can run in both nodejs and browser, the distribution should specify where it is intended to run.
	RuntimeEnvironments []RuntimeEnvironment `yaml:"runtimeEnvironments"`

	// A list of frameworks this distribution targets (can be left empty)
	Frameworks []Framework `yaml:"frameworks"`

	// Free text description of the distribution, what it includes, it's use cases, etc.
	Description string `yaml:"description"`

	// categories of environments variables that need to be set in the application runtime
	// to enable the distribution.
	EnvironmentVariables EnvironmentVariables `yaml:"environmentVariables,omitempty"`

	// Metadata and properties of the runtime agent that is used to enable the distribution.
	// Can be nil in case no runtime agent is required.
	RuntimeAgent *RuntimeAgent `yaml:"runtimeAgent,omitempty"`

	// if true, the distro receives it's configuration as environment variables.
	// it means the distro does not support opamp and not configurable via ebpf.
	// these pods will require a restart to apply the new configuration.
	// used for java as temporary solution until we have a better way to configure the agent.
	ConfigAsEnvVars bool `yaml:"configAsEnvVars,omitempty"`

	// document support for metrics produced directly from the runtime
	AgentMetrics *AgentMetrics `yaml:"agentMetrics,omitempty"`

	// document support by this distro for trace features
	Traces *Traces `yaml:"traces,omitempty"`
}
