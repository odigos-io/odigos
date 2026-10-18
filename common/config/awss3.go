package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/odigos-io/odigos/common"
)

const (
	s3BucketKey          = "S3_BUCKET"
	s3RegionKey          = "S3_REGION"
	s3PartitionKey       = "S3_PARTITION"
	s3PartitionFormatKey = "S3_PARTITION_FORMAT"
	s3Marshaller         = "S3_MARSHALER"
	s3EndpointKey        = "S3_ENDPOINT"
	s3PathStyleKey       = "S3_FORCE_PATH_STYLE"
	s3AccessKey          = "S3_ACCESS_KEY_ID"
	s3SecretKey          = "S3_SECRET_ACCESS_KEY"
	s3CAKey              = "S3_CA_PEM"
	s3CompressionKey     = "S3_COMPRESSION"
	s3AuthModeKey        = "S3_AUTH_MODE"
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
	return configureS3(dest, currentConfig, "awss3")
}

func configureS3(dest ExporterConfigurer, currentConfig *Config, prefix string) ([]string, error) {
	if !isLoggingEnabled(dest) && !isTracingEnabled(dest) && !isMetricsEnabled(dest) {
		return nil, errors.New("no metrics, logs or traces enabled, gateway will not be configured for AWS S3")
	}

	bucket, ok := dest.GetConfig()[s3BucketKey]
	if !ok || bucket == "" {
		return nil, ErrS3BucketNotSpecified
	}

	region, ok := dest.GetConfig()[s3RegionKey]
	if !ok || region == "" {
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

	uploader := GenericMap{"region": region, "s3_bucket": bucket, "s3_partition_format": partitionFormat}
	if err := configureS3Uploader(dest, uploader); err != nil {
		return nil, err
	}

	exporterName := "awss3/" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{"s3uploader": uploader, "marshaler": marshaler}

	var pipelineNames []string
	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/" + prefix + "-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/" + prefix + "-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/" + prefix + "-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}

func configureS3Uploader(dest ExporterConfigurer, uploader GenericMap) error {
	if endpoint := dest.GetConfig()[s3EndpointKey]; endpoint != "" {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Host == "" ||
			(parsed.Scheme != "https" && parsed.Scheme != "http") ||
			parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.New("S3_ENDPOINT must be an HTTP or HTTPS URL without credentials, query or fragment")
		}
		uploader["endpoint"] = endpoint
		uploader["disable_ssl"] = parsed.Scheme == "http"
	}
	if value, ok := dest.GetConfig()[s3PathStyleKey]; ok {
		pathStyle, err := strconv.ParseBool(value)
		if err != nil {
			return errors.New("S3_FORCE_PATH_STYLE must be a boolean")
		}
		uploader["s3_force_path_style"] = pathStyle
	} else if dest.GetType() == common.OpenStackSwiftDestinationType {
		uploader["s3_force_path_style"] = true
	}
	if dest.GetType() == common.OpenStackSwiftDestinationType || dest.GetConfig()[s3AuthModeKey] == "static" {
		uploader["use_static_credentials"] = true
		uploader["access_key_id"] = SecretEnvPlaceholder(s3AccessKey, dest)
		uploader["secret_access_key"] = SecretEnvPlaceholder(s3SecretKey, dest)
	}
	if ca := dest.GetConfig()[s3CAKey]; ca != "" {
		uploader["tls"] = GenericMap{"ca_pem": ca}
	}
	if compression := dest.GetConfig()[s3CompressionKey]; compression != "" && compression != "none" {
		if compression != "gzip" && compression != "zstd" {
			return errors.New("S3_COMPRESSION must be none, gzip or zstd")
		}
		uploader["compression"] = compression
	}
	if dest.GetType() == common.OpenStackSwiftDestinationType {
		uploader["request_checksum_calculation"] = "when_required"
	}
	return nil
}
