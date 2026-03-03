package consts

import (
	"errors"
	"time"
)

const (
	CurrentNamespaceEnvVar              = "CURRENT_NS"
	OdigosVersionEnvVarName             = "ODIGOS_VERSION"
	OdigosComponentDeploymentNameEnvVar = "ODIGOS_COMPONENT_DEPLOYMENT_NAME"
	OdigosTierEnvVarName                = "ODIGOS_TIER"
	DefaultOdigosNamespace              = "odigos-system"
	OdigosConfigurationName             = "odigos-configuration"
	// Deprecated: only used for migrations
	OdigosLegacyConfigName      = "odigos-config"
	OdigosEffectiveConfigName   = "effective-config"
	OdigosRemoteConfigName      = "odigos-remote-config"
	OdigosLocalUiConfigName     = "odigos-local-ui-config"
	OdigosConfigurationFileName = "config.yaml"
	OTLPPort                    = 4317
	OTLPHttpPort                = 4318

	// DefaultDataStream is the default data stream name used for telemetry data.
	DefaultDataStream = "default"

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

	GoOffsetsPublicURL = "https://storage.googleapis.com/odigos-cloud/offset_results_min.json"

	LdPreloadEnvVarName = "LD_PRELOAD"
	OdigosLoaderDirName = "loader"
	OdigosLoaderName    = "loader.so"

	// name of the secret that contains the oidc client secret
	OidcSecretName = "odigos-oidc"

	ServiceGraphConnectorName = "servicegraph"
	ServiceGraphEndpointPort  = 9090

	// Custom attribute to distinguish workload types that share the same semconv key (e.g., DeploymentConfig uses k8s.deployment.name)
	// This allows the UI to distinguish between DeploymentConfig and Deployment, and construct the correct Source workload.
	// Since DeploymentConfig uses k8s.deployment.name as the semconv key, we need to add this attribute to the list of attributes to be collected.
	// This is needed to distinguish between workloads that share the same semconv key
	// (e.g., DeploymentConfig uses k8s.deployment.name)
	OdigosWorkloadKindAttribute = "odigos.workload.kind"

	// Custom attribute which specifies the odigos workload/source name.
	// For Deployments this will be the deployment name, for StatefulSets the statefulset name, etc'
	// See Source CRD workload field for more details.
	OdigosWorkloadNameAttribute = "odigos.workload.name"
)

// Odigos config properties
const (
	TelemetryEnabledProperty           = "telemetry-enabled"
	OpenshiftEnabledProperty           = "openshift-enabled"
	PspProperty                        = "psp"
	SkipWebhookIssuerCreationProperty  = "skip-webhook-issuer-creation"
	AllowConcurrentAgentsProperty      = "allow-concurrent-agents"
	ImagePrefixProperty                = "image-prefix"
	UiModeProperty                     = "ui-mode"
	UiPaginationLimitProperty          = "ui-pagination-limit"
	UiRemoteUrlProperty                = "ui-remote-url"
	CentralBackendURLProperty          = "central-backend-url"
	ClusterNameProperty                = "cluster-name"
	IgnoredNamespacesProperty          = "ignored-namespaces"
	IgnoredContainersProperty          = "ignored-containers"
	MountMethodProperty                = "mount-method"
	CustomContainerRuntimeSocketPath   = "custom-container-runtime-socket-path"
	K8sNodeLogsDirectory               = "k8s-node-logs-directory"
	UserInstrumentationEnvsProperty    = "user-instrumentation-envs"
	AgentEnvVarsInjectionMethod        = "agent-env-vars-injection-method"
	NodeSelectorProperty               = "node-selector"
	KarpenterEnabledProperty           = "karpenter-enabled"
	RollbackDisabledProperty           = "instrumentation-auto-rollback-disabled"
	RollbackGraceTimeProperty          = "instrumentation-auto-rollback-grace-time"
	RollbackStabilityWindow            = "instrumentation-auto-rollback-stability-window"
	AutomaticRolloutDisabledProperty   = "automatic-rollout-disabled"
	OidcTenantUrlProperty              = "oidc-tenant-url"
	OidcClientIdProperty               = "oidc-client-id"
	OidcClientSecretProperty           = "oidc-client-secret"
	OdigletHealthProbeBindPortProperty = "odiglet-health-probe-bind-port"
	ServiceGraphDisabledProperty       = "service-graph-disabled"
	GoAutoOffsetsCronProperty          = "go-auto-offsets-cron"
	GoAutoOffsetsModeProperty          = "go-auto-offsets-mode"
	ClickhouseJsonTypeEnabledProperty  = "clickhouse-json-type-enabled"
	AllowedTestConnectionHostsProperty = "allowed-test-connection-hosts"
	EnableDataCompressionProperty      = "enable-data-compression"
	ResourceSizePresetProperty         = "resource-size-preset"
	WaspEnabledProperty                = "wasp-enabled"
)

var (
	ErrorPodsNotFound = errors.New("could not find a ready pod")
)

// Agents related consts
var (
	OtelLogsExporter            = "OTEL_LOGS_EXPORTER"
	OtelMetricsExporter         = "OTEL_METRICS_EXPORTER"
	OtelTracesExporter          = "OTEL_TRACES_EXPORTER"
	OtelExporterEndpointEnvName = "OTEL_EXPORTER_OTLP_ENDPOINT"
	// Python related ones
	OpampServerHostEnvName = "ODIGOS_OPAMP_SERVER_HOST"
	OpAMPPort              = 4320
)

// Odigos Central related consts
const (
	DefaultOdigosCentralNamespace = "odigos-central"
)

// Karpenter related consts
const (
	KarpenterStartupTaintKey = "odigos.io/needs-init"
)

// Processor related consts
const (
	GenericBatchProcessorConfigKey = "batch/generic-batch-processor"
	SmallBatchesProcessor          = "batch/small-batches"
	GroupByTraceProcessorV2        = "groupbytrace/samplingv2"
)

// Auto rollback related consts
var (
	DefaultAutoRollbackGraceTime       = 5 * time.Minute
	DefaultAutoRollbackStabilityWindow = 1 * time.Hour
)

// UnixFD related consts
const (
	ExchangeDir = "/var/exchange"
)
