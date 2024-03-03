package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	genericOtlpUrlKey = "OTLP_GRPC_ENDPOINT"
)

type GenericOTLP struct{}

func (g *GenericOTLP) DestType() common.DestinationType {
	return common.GenericOTLPDestinationType
}

func (g *GenericOTLP) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	url, exists := dest.Spec.Data[genericOtlpUrlKey]
	if !exists {
		log.Log.V(0).Info("Generic OTLP gRPC endpoint not specified, gateway will not be configured for otlp")
		return
	}

	grpcEndpoint, err := parseUnencryptedOtlpGrpcUrl(url)
	if err != nil {
		log.Log.V(0).Error(err, "otlp endpoint invalid, gateway will not be configured for otlp")
		return
	}

	genericOtlpExporterName := "otlp/generic-" + dest.Name
	currentConfig.Exporters[genericOtlpExporterName] = commonconf.GenericMap{
		"endpoint": grpcEndpoint,
		"tls": commonconf.GenericMap{
			"insecure": true,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/generic-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/generic-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/generic-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
	}
}
