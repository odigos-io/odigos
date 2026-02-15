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
	MetricsConsumer *collectormetrics.OdigosMetricsConsumer
	Logger          logr.Logger
	PromAPI         v1.API
	// K8sCacheClient is the controller-runtime client that reads from the informer cache (fast, no API round-trip).
	// Use this in resolvers for read-only access to cluster state.
	K8sCacheClient client.Client
}
