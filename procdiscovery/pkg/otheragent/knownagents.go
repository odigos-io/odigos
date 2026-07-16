package otheragent

import "github.com/odigos-io/odigos/common"

// Agent names — surfaced in the OtherAgentDetected condition.
const (
	DatadogAgentName       = "Datadog Agent"
	NewRelicAgentName      = "New Relic Agent"
	DynatraceAgentName     = "Dynatrace Agent"
	OpenTelemetryAgentName = "OpenTelemetry Agent"
	GrafanaOtelAgentName   = "Grafana OpenTelemetry Agent"
	SplunkOtelAgentName    = "Splunk OpenTelemetry Agent"
	AWSDistroOtelAgentName = "AWS Distro for OpenTelemetry Agent"
	ElasticAPMAgentName    = "Elastic APM Agent"
	AppDynamicsAgentName   = "AppDynamics Agent"
	InstanaAgentName       = "Instana Agent"
)

// Vendor-specific .NET CLR profiler GUIDs.
const (
	datadogDotNetProfilerGUID  = "846F5F1C"
	newRelicDotNetProfilerGUID = "36032161-FFC0-4B61-B559-F6C5D41BAE5A"
	otelDotNetProfilerGUID     = "918728DD-259F-4A6A-AC2B-B85E1B658318"
)

// KnownAgents lists conclusive markers that an instrumentation agent is loaded
// in a process. A single match means detected.
var KnownAgents = []KnownAgent{
	// Datadog
	{Name: DatadogAgentName, Language: common.DotNetProgrammingLanguage, Signal: EnvValueContains,
		Key: "CORECLR_PROFILER", Match: datadogDotNetProfilerGUID},
	{Name: DatadogAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "dd-java-agent"},
	{Name: DatadogAgentName, Language: common.JavaProgrammingLanguage, Signal: EnvValueContains,
		Key: "JAVA_TOOL_OPTIONS", Match: "dd-java-agent"},
	{Name: DatadogAgentName, Language: common.JavascriptProgrammingLanguage, Signal: EnvValueContains, Key: "NODE_OPTIONS", Match: "dd-trace"},
	{Name: DatadogAgentName, Language: common.RubyProgrammingLanguage, Signal: EnvValueContains, Key: "RUBYOPT", Match: "datadog"},
	{Name: DatadogAgentName, Language: common.RubyProgrammingLanguage, Signal: EnvValueContains, Key: "RUBYOPT", Match: "ddtrace"},
	{Name: DatadogAgentName, Language: common.PythonProgrammingLanguage, Signal: CmdlineContains, Match: "ddtrace-run"},
	{Name: DatadogAgentName, Language: common.PhpProgrammingLanguage, Signal: LibLoaded, Match: "ddtrace.so"},
	{Name: DatadogAgentName, Signal: EnvValueContains, Key: "LD_PRELOAD", Match: "datadog-apm-inject"},
	{Name: DatadogAgentName, Signal: EnvPresent, Key: "DD_INJECTION_ENABLED"},
	{Name: DatadogAgentName, Signal: EnvPresent, Key: "DD_TRACE_AGENT_URL"},

	// New Relic
	{Name: NewRelicAgentName, Signal: EnvPresent, Key: "NEW_RELIC_CONFIG_FILE"},
	{Name: NewRelicAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "newrelic.jar"},
	{Name: NewRelicAgentName, Language: common.JavascriptProgrammingLanguage, Signal: EnvValueContains, Key: "NODE_OPTIONS", Match: "newrelic"},
	{Name: NewRelicAgentName, Language: common.DotNetProgrammingLanguage, Signal: EnvValueContains,
		Key: "CORECLR_PROFILER", Match: newRelicDotNetProfilerGUID},
	{Name: NewRelicAgentName, Language: common.PhpProgrammingLanguage, Signal: LibLoaded, Match: "newrelic.so"},

	// Dynatrace
	{Name: DynatraceAgentName, Signal: EnvPresent, Key: "DT_DYNAMIZER_TARGET_EXE"},
	{Name: DynatraceAgentName, Signal: EnvValueContains, Key: "LD_PRELOAD", Match: "/dynatrace/"},
	{Name: DynatraceAgentName, Signal: LibLoaded, Match: "liboneagent"},

	// OpenTelemetry
	{Name: OpenTelemetryAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "opentelemetry-javaagent"},
	{Name: OpenTelemetryAgentName, Language: common.JavaProgrammingLanguage, Signal: EnvValueContains,
		Key: "JAVA_TOOL_OPTIONS", Match: "opentelemetry-javaagent"},
	{Name: OpenTelemetryAgentName, Language: common.JavascriptProgrammingLanguage, Signal: EnvValueContains,
		Key: "NODE_OPTIONS", Match: "@opentelemetry/auto-instrumentations-node"},
	{Name: OpenTelemetryAgentName, Language: common.PythonProgrammingLanguage, Signal: CmdlineContains, Match: "opentelemetry-instrument"},
	{Name: OpenTelemetryAgentName, Language: common.DotNetProgrammingLanguage, Signal: EnvValueContains,
		Key: "CORECLR_PROFILER", Match: otelDotNetProfilerGUID},

	// Grafana OpenTelemetry
	{Name: GrafanaOtelAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "grafana-opentelemetry-java"},
	{Name: GrafanaOtelAgentName, Language: common.JavaProgrammingLanguage, Signal: EnvValueContains,
		Key: "JAVA_TOOL_OPTIONS", Match: "grafana-opentelemetry-java"},

	// Splunk / SignalFx OpenTelemetry
	{Name: SplunkOtelAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "splunk-otel-javaagent"},

	// AWS Distro for OpenTelemetry
	{Name: AWSDistroOtelAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "aws-opentelemetry-agent"},

	// Elastic APM
	{Name: ElasticAPMAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "elastic-apm-agent"},

	// AppDynamics
	{Name: AppDynamicsAgentName, Signal: EnvPresent, Key: "APPDYNAMICS_CONTROLLER_HOST_NAME"},
	{Name: AppDynamicsAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "appdynamics"},

	// Instana
	{Name: InstanaAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "instana"},
}
