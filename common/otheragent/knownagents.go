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

// Environment variable names inspected by the env-based entries in KnownAgents.
const (
	envCORECLRProfiler           = "CORECLR_PROFILER"
	envJavaToolOptions           = "JAVA_TOOL_OPTIONS"
	envNodeOptions               = "NODE_OPTIONS"
	envRubyOpt                   = "RUBYOPT"
	envLDPreload                 = "LD_PRELOAD"
	envDDInjectionEnabled        = "DD_INJECTION_ENABLED"
	envDDTraceAgentURL           = "DD_TRACE_AGENT_URL"
	envNewRelicConfigFile        = "NEW_RELIC_CONFIG_FILE"
	envDTDynamizerTargetExe      = "DT_DYNAMIZER_TARGET_EXE"
	envAppDynamicsControllerHost = "APPDYNAMICS_CONTROLLER_HOST_NAME"
)

// AgentDetectionEnvKeys are the environment variables inspected by the env-based
// entries in KnownAgents. Consumers collect these into the process env so that
// detection can read them.
var AgentDetectionEnvKeys = map[string]struct{}{
	envCORECLRProfiler:           {},
	envJavaToolOptions:           {},
	envNodeOptions:               {},
	envRubyOpt:                   {},
	envLDPreload:                 {},
	envDDInjectionEnabled:        {},
	envDDTraceAgentURL:           {},
	envNewRelicConfigFile:        {},
	envDTDynamizerTargetExe:      {},
	envAppDynamicsControllerHost: {},
}

// KnownAgents lists conclusive markers that an instrumentation agent is loaded
// in a process. A single match means detected.
var KnownAgents = []KnownAgent{
	// Datadog
	{Name: DatadogAgentName, Language: common.DotNetProgrammingLanguage, Signal: EnvValueContains,
		Key: envCORECLRProfiler, Match: datadogDotNetProfilerGUID},
	{Name: DatadogAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "dd-java-agent"},
	{Name: DatadogAgentName, Language: common.JavaProgrammingLanguage, Signal: EnvValueContains,
		Key: envJavaToolOptions, Match: "dd-java-agent"},
	{Name: DatadogAgentName, Language: common.JavascriptProgrammingLanguage, Signal: EnvValueContains, Key: envNodeOptions, Match: "dd-trace"},
	{Name: DatadogAgentName, Language: common.RubyProgrammingLanguage, Signal: EnvValueContains, Key: envRubyOpt, Match: "datadog"},
	{Name: DatadogAgentName, Language: common.RubyProgrammingLanguage, Signal: EnvValueContains, Key: envRubyOpt, Match: "ddtrace"},
	{Name: DatadogAgentName, Language: common.PythonProgrammingLanguage, Signal: CmdlineContains, Match: "ddtrace-run"},
	{Name: DatadogAgentName, Language: common.PhpProgrammingLanguage, Signal: LibLoaded, Match: "ddtrace.so"},
	{Name: DatadogAgentName, Signal: EnvValueContains, Key: envLDPreload, Match: "datadog-apm-inject"},
	{Name: DatadogAgentName, Signal: EnvPresent, Key: envDDInjectionEnabled},
	{Name: DatadogAgentName, Signal: EnvPresent, Key: envDDTraceAgentURL},

	// New Relic
	{Name: NewRelicAgentName, Signal: EnvPresent, Key: envNewRelicConfigFile},
	{Name: NewRelicAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "newrelic.jar"},
	{Name: NewRelicAgentName, Language: common.JavascriptProgrammingLanguage, Signal: EnvValueContains, Key: envNodeOptions, Match: "newrelic"},
	{Name: NewRelicAgentName, Language: common.DotNetProgrammingLanguage, Signal: EnvValueContains,
		Key: envCORECLRProfiler, Match: newRelicDotNetProfilerGUID},
	{Name: NewRelicAgentName, Language: common.PhpProgrammingLanguage, Signal: LibLoaded, Match: "newrelic.so"},

	// Dynatrace
	{Name: DynatraceAgentName, Signal: EnvPresent, Key: envDTDynamizerTargetExe},
	{Name: DynatraceAgentName, Signal: EnvValueContains, Key: envLDPreload, Match: "/dynatrace/"},
	{Name: DynatraceAgentName, Signal: LibLoaded, Match: "liboneagent"},

	// OpenTelemetry
	{Name: OpenTelemetryAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "opentelemetry-javaagent"},
	{Name: OpenTelemetryAgentName, Language: common.JavaProgrammingLanguage, Signal: EnvValueContains,
		Key: envJavaToolOptions, Match: "opentelemetry-javaagent"},
	{Name: OpenTelemetryAgentName, Language: common.JavascriptProgrammingLanguage, Signal: EnvValueContains,
		Key: envNodeOptions, Match: "@opentelemetry/auto-instrumentations-node"},
	{Name: OpenTelemetryAgentName, Language: common.PythonProgrammingLanguage, Signal: CmdlineContains, Match: "opentelemetry-instrument"},
	{Name: OpenTelemetryAgentName, Language: common.DotNetProgrammingLanguage, Signal: EnvValueContains,
		Key: envCORECLRProfiler, Match: otelDotNetProfilerGUID},

	// Grafana OpenTelemetry
	{Name: GrafanaOtelAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "grafana-opentelemetry-java"},
	{Name: GrafanaOtelAgentName, Language: common.JavaProgrammingLanguage, Signal: EnvValueContains,
		Key: envJavaToolOptions, Match: "grafana-opentelemetry-java"},

	// Splunk / SignalFx OpenTelemetry
	{Name: SplunkOtelAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "splunk-otel-javaagent"},

	// AWS Distro for OpenTelemetry
	{Name: AWSDistroOtelAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "aws-opentelemetry-agent"},

	// Elastic APM
	{Name: ElasticAPMAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "elastic-apm-agent"},

	// AppDynamics
	{Name: AppDynamicsAgentName, Signal: EnvPresent, Key: envAppDynamicsControllerHost},
	{Name: AppDynamicsAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "appdynamics"},

	// Instana
	{Name: InstanaAgentName, Language: common.JavaProgrammingLanguage, Signal: CmdlineContains, Match: "instana"},
}
