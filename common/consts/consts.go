package consts

import "errors"

const (
	CurrentNamespaceEnvVar         = "CURRENT_NS"
	DefaultNamespace               = "odigos-system"
	DefaultOdigosConfigurationName = "odigos-config"
	OTLPPort                       = 4317
	OTLPHttpPort                   = 4318
	OdigosInstrumentationLabel     = "odigos-instrumentation"
	InstrumentationEnabled         = "enabled"
	InstrumentationDisabled        = "disabled"
	GolangInstrumentationImage     = "keyval/otel-go-agent:v0.6.5"
	OdigosReportedNameAnnotation   = "odigos.io/reported-name"
	EbpfInstrumentationAnnotation  = "instrumentation.odigos.io/ebpf" // deprecated.
)

var (
	PodsNotFoundErr = errors.New("could not find a ready pod")
)
