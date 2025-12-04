package graph

import (
	"github.com/go-logr/logr"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// This file will not be regenerated automatically.
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	MetricsConsumer *collectormetrics.OdigosMetricsConsumer
	Logger          logr.Logger
	PromAPI         v1.API
}
