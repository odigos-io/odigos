package config

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/odigos-io/odigos/common"
)

const (
	s3BucketKey          = "S3_BUCKET"
	s3RegionKey          = "S3_REGION"
	s3PartitionKey       = "S3_PARTITION"
	s3PartitionFormatKey = "S3_PARTITION_FORMAT"
	s3Marshaller         = "S3_MARSHALER"
)

var (
	ErrS3BucketNotSpecified = fmt.Errorf("s3 bucket not specified")
	ErrS3RegionNotSpecified = fmt.Errorf("s3 region not specified")
)

type AWSS3 struct{}

func (s *AWSS3) DestType() common.DestinationType {
	return common.AWSS3DestinationType
}

func (s *AWSS3) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	return configureS3(dest, currentConfig, false)
}

func configureS3(dest ExporterConfigurer, currentConfig *Config, spaces bool) ([]string, error) {
	if !isLoggingEnabled(dest) && !isTracingEnabled(dest) && !isMetricsEnabled(dest) {
		return nil, errors.New("no metrics, logs or traces enabled, gateway will not be configured for AWS S3")
	}

	bucket, ok := dest.GetConfig()[s3BucketKey]
	if !ok {
		return nil, ErrS3BucketNotSpecified
	}

	region, ok := dest.GetConfig()[s3RegionKey]
	if !ok {
		return nil, ErrS3RegionNotSpecified
	}

	// Handle partition format - support both old s3_partition and new s3_partition_format
	var partitionFormat string

	// Check for new s3_partition_format first
	if format, ok := dest.GetConfig()[s3PartitionFormatKey]; ok {
		partitionFormat = format
	} else {
		// Fall back to old s3_partition for backward compatibility
		partition, ok := dest.GetConfig()[s3PartitionKey]
		if !ok {
			partition = "minute"
		}

		// Convert old partition values to new format
		switch partition {
		case "minute":
			partitionFormat = "year=%Y/month=%m/day=%d/hour=%H/minute=%M"
		case "hour":
			partitionFormat = "year=%Y/month=%m/day=%d/hour=%H"
		default:
			return nil, errors.New("invalid partition specified, gateway will not be configured for AWS S3")
		}
	}

	marshaler, ok := dest.GetConfig()[s3Marshaller]
	if !ok {
		marshaler = "otlp_json"
	}
	if marshaler != "otlp_json" && marshaler != "otlp_proto" {
		return nil, errors.New("invalid marshaller specified, gateway will not be configured for AWS S3")
	}

	exporterName := "awss3/" + dest.GetID()
	exporterConfig := GenericMap{
		"s3uploader": GenericMap{
			"region":              region,
			"s3_bucket":           bucket,
			"s3_partition_format": partitionFormat,
		},
		"marshaler": marshaler,
	}

	uploader := exporterConfig["s3uploader"].(GenericMap)
	for field, key := range map[string]string{"S3_ENDPOINT": "endpoint", "S3_COMPRESSION": "compression", "S3_PREFIX": "s3_prefix"} {
		if value := dest.GetConfig()[field]; value != "" {
			uploader[key] = value
		}
	}
	if value := dest.GetConfig()["S3_FORCE_PATH_STYLE"]; value != "" {
		pathStyle, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid S3_FORCE_PATH_STYLE: %w", err)
		}
		uploader["s3_force_path_style"] = pathStyle
	}
	if spaces {
		uploader["access_key_id"] = SecretEnvPlaceholder("S3_ACCESS_KEY_ID", dest)
		uploader["secret_access_key"] = SecretEnvPlaceholder("S3_SECRET_ACCESS_KEY", dest)
		uploader["checksum_when_required"] = true
		uploader["unique_key_func_name"] = "uuidv7"
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	var pipelineNames []string
	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/awss3-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/awss3-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/awss3-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
