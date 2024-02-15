package config

import (
	"fmt"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	s3BucketKey    = "S3_BUCKET"
	s3RegionKey    = "S3_REGION"
	s3PartitionKey = "S3_PARTITION"
	s3Marshaller   = "S3_MARSHALER"
)

var (
	ErrS3BucketNotSpecified = fmt.Errorf("s3 bucket not specified")
	ErrS3RegionNotSpecified = fmt.Errorf("s3 region not specified")
)

type AWSS3 struct{}

func (s *AWSS3) DestType() common.DestinationType {
	return common.AWSS3DestinationType
}

func (s *AWSS3) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	if !isLoggingEnabled(dest) && !isTracingEnabled(dest) && !isMetricsEnabled(dest) {
		log.Log.V(0).Info("No metrics, logs or traces enabled, gateway will not be configured for AWS S3")
		return
	}

	bucket, ok := dest.Spec.Data[s3BucketKey]
	if !ok {
		ctrl.Log.Error(ErrS3BucketNotSpecified, "s3 bucket not specified")
		return
	}

	region, ok := dest.Spec.Data[s3RegionKey]
	if !ok {
		ctrl.Log.Error(ErrS3RegionNotSpecified, "s3 region not specified")
		return
	}

	partition, ok := dest.Spec.Data[s3PartitionKey]
	if !ok {
		partition = "minute"
	}
	if partition != "minute" && partition != "hour" {
		log.Log.V(0).Info("Invalid partition specified, gateway will not be configured for AWS S3")
		return
	}

	marshaler, ok := dest.Spec.Data[s3Marshaller]
	if !ok {
		marshaler = "otlp_json"
	}
	if marshaler != "otlp_json" && marshaler != "otlp_proto" {
		log.Log.V(0).Info("Invalid marshaller specified, gateway will not be configured for AWS S3")
		return
	}

	exporterName := "awss3/" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"s3uploader": commonconf.GenericMap{
			"region":       region,
			"s3_bucket":    bucket,
			"s3_partition": partition,
		},
		"marshaler": marshaler,
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/awss3-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/awss3-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/awss3-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}
}
