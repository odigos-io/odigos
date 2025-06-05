package metrics

import (
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	mstricapi "go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	controllermetric "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// NewMeterProviderForController creates a new MeterProvider for the controller with the given resource.
// It uses controller-runtime's prometheus metrics registry, and sets up the Otel MeterProvider to integrate with it.
// The MeterProvider is configured with the provided resource.
//
// Consumers of the created MeterProvider can create different meters and create custom instruments for custom metrics.
//
// This should only be called in a controller-runtime service.
func NewMeterProviderForController(res *resource.Resource) (mstricapi.MeterProvider, error) {
	e, err := otelprometheus.New(
		otelprometheus.WithRegisterer(controllermetric.Registry),
	)
	if err != nil {
		return nil, err
	}

	provider := metricsdk.NewMeterProvider(
		metricsdk.WithReader(e),
		metricsdk.WithResource(res),
	)

	return provider, nil
}
