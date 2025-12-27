package graph

import (
	"github.com/go-logr/logr"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This file will not be regenerated automatically.
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	// allows the resolver to access the k8s cache client to fetch k8s resources efficiently.
	k8sCacheClient client.Client

	MetricsConsumer *collectormetrics.OdigosMetricsConsumer
	Logger          logr.Logger
	PromAPI         v1.API
}
