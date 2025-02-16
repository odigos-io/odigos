package consts

import (
	"errors"
)

const (
	CurrentNamespaceEnvVar      = "CURRENT_NS"
	OdigosVersionEnvVarName     = "ODIGOS_VERSION"
	OdigosTierEnvVarName        = "ODIGOS_TIER"
	DefaultOdigosNamespace      = "odigos-system"
	OdigosConfigurationName     = "odigos-config"
	OdigosEffectiveConfigName   = "effective-config"
	OdigosConfigurationFileName = "config.yaml"
	OTLPPort                    = 4317
	OTLPHttpPort                = 4318
	PprofOdigosPort             = 6060

	// Deprecated: Sources are used to mark workloads for instrumentation.
	OdigosInstrumentationLabel = "odigos-instrumentation"

	// Deprecated: Sources are used to mark workloads for instrumentation.
	InstrumentationEnabled = "enabled"

	// Deprecated: Sources are used to mark workloads for instrumentation.
	InstrumentationDisabled = "disabled"

	// Deprecated: reported name is set via the Source CR.
	OdigosReportedNameAnnotation = "odigos.io/reported-name"
	RolloutTriggerAnnotation     = "rollout-trigger"

	// GatewayMaxConnectionAge and GatewayMaxConnectionAgeGrace are the default values for the gateway collector.
	GatewayMaxConnectionAge      = "15s"
	GatewayMaxConnectionAgeGrace = "2s"

	// Used to store the original value of the environment variable in the pod manifest.
	// This is used to restore the original value when an instrumentation is removed
	// or odigos is uninstalled.
	// Should only be used for environment variables that are modified by odigos.
	ManifestEnvOriginalValAnnotation = "odigos.io/manifest-env-original-val"

	// Used to label instrumentation instances by the corresponding
	// instrumented app for better query performance.
	InstrumentedAppNameLabel = "instrumented-app"

	// CRD types
	InstrumentationConfig   = "InstrumentationConfig"
	InstrumentationInstance = "InstrumentationInstance"
	Destination             = "Destination"
)

var (
	ErrorPodsNotFound = errors.New("could not find a ready pod")
)

// Agents related consts
var (
	OtelExporterEndpointEnvName = "OTEL_EXPORTER_OTLP_ENDPOINT"
	// Python related ones
	OtelPythonConfiguratorEnvName      = "OTEL_PYTHON_CONFIGURATOR"
	OtelPythonOSSConfiguratorEnvValue  = "odigos-python-configurator"
	OtelPythonEBPFConfiguratorEnvValue = "ebpf"
	OpampServerHostEnvName             = "ODIGOS_OPAMP_SERVER_HOST"
	OpAMPPort                          = 4320
)
