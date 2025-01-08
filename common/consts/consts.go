package consts

import (
	"errors"
)

const (
	CurrentNamespaceEnvVar       = "CURRENT_NS"
	DefaultOdigosNamespace       = "odigos-system"
	OdigosConfigurationName      = "odigos-config"
	OdigosConfigurationFileName  = "config.yaml"
	OTLPPort                     = 4317
	OTLPHttpPort                 = 4318
	PprofOdigosPort              = 6060
	OdigosInstrumentationLabel   = "odigos-instrumentation"
	InstrumentationEnabled       = "enabled"
	InstrumentationDisabled      = "disabled"
	OdigosNamespaceAnnotation    = "odigos.io/workload-namespace"
	OdigosWorkloadKindAnnotation = "odigos.io/workload-kind"
	OdigosWorkloadNameAnnotation = "odigos.io/workload-name"
	OdigosWorkloadExcludedLabel  = "odigos.io/workload-excluded"
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

var (
	SystemNamespaces  = []string{DefaultOdigosNamespace, "kube-system", "local-path-storage", "istio-system", "linkerd", "kube-node-lease"}
	IgnoredContainers = []string{"istio-proxy", "vault-agent", "filebeat", "linkerd-proxy", "fluentd", "akeyless-init"}
)
