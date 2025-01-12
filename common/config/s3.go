package config

import (
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
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

func (s *AWSS3) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	if !isLoggingEnabled(dest) && !isTracingEnabled(dest) && !isMetricsEnabled(dest) {
		return errors.New("No metrics, logs or traces enabled, gateway will not be configured for AWS S3")
	}

	bucket, ok := dest.GetConfig()[s3BucketKey]
	if !ok {
		return ErrS3BucketNotSpecified
	}

	region, ok := dest.GetConfig()[s3RegionKey]
	if !ok {
		return ErrS3RegionNotSpecified
	}

	partition, ok := dest.GetConfig()[s3PartitionKey]
	if !ok {
		partition = "minute"
	}
	if partition != "minute" && partition != "hour" {
		return errors.New("Invalid partition specified, gateway will not be configured for AWS S3")
	}

	marshaler, ok := dest.GetConfig()[s3Marshaller]
	if !ok {
		marshaler = "otlp_json"
	}
	if marshaler != "otlp_json" && marshaler != "otlp_proto" {
		return errors.New("Invalid marshaller specified, gateway will not be configured for AWS S3")
	}

	exporterName := "awss3/" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"s3uploader": GenericMap{
			"region":       region,
			"s3_bucket":    bucket,
			"s3_partition": partition,
		},
		"marshaler": marshaler,
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/awss3-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/awss3-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/awss3-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
