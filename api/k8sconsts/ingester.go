package k8sconsts

const (
	JaegerPrefix  = "jaegertracing"
	JaegerImage   = "jaeger"
	JaegerVersion = "2.1.0"

	IngesterImage              = "odigos-ingester"
	IngesterImageUBI9          = "odigos-ingester-ubi9"
	IngesterServiceAccountName = "odigos-ingester"
	IngesterServiceName        = "ingester"
	IngesterDeploymentName     = "odigos-ingester"
	IngesterAppLabelValue      = "odigos-ingester"
	IngesterContainerName      = "ingester"
	IngesterApiPort            = 16686
)
