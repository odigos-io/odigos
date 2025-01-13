package graph

import (
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
)

// This file will not be regenerated automatically.
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	MetricsConsumer *collectormetrics.OdigosMetricsConsumer
}
