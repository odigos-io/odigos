package config

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/destinations"
)

const (
	gcpProjectIdKey              = "GCP_PROJECT_ID"
	gcpBillingProjectIdKey       = "GCP_BILLING_PROJECT"
	gcpApplicationCredentialsKey = "GCP_APPLICATION_CREDENTIALS"
	gcpCredentialsMountPath      = "/secrets"
)

type GoogleCloud struct{}

func (g *GoogleCloud) DestType() destinations.DestinationType {
	return destinations.GoogleCloudDestinationType
}

func (g *GoogleCloud) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if isTracingEnabled(dest) {
		exporterName := "googlecloud/" + dest.GetID()
		currentConfig.Exporters[exporterName] = struct{}{}

		tracesPipelineName := "traces/googlecloud-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}
	var pipelineNames []string
	if isLoggingEnabled(dest) {
		exporterName := "googlecloud/" + dest.GetID()
		currentConfig.Exporters[exporterName] = GenericMap{
			"log": GenericMap{
				"default_log_name": "opentelemetry.io/collector-exported-log",
			},
		}

		logsPipelineName := "logs/googlecloud-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}

type GoogleCloudOTLP struct{}

func (g *GoogleCloudOTLP) DestType() destinations.DestinationType {
	return destinations.GoogleCloudOTLPDestinationType
}

func (g *GoogleCloudOTLP) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if isTracingEnabled(dest) {
		exporterName := "otlphttp/gcp-" + dest.GetID()
		extensionName := "googleclientauth/" + dest.GetID()
		processorName := "resource/gcp-" + dest.GetID()
		config := dest.GetConfig()
		exporterConfig := GenericMap{
			"encoding": "proto",
			"endpoint": "https://telemetry.googleapis.com",
			"auth": GenericMap{
				"authenticator": extensionName,
			},
		}
		currentConfig.Exporters[exporterName] = exporterConfig

		processorConfig := GenericMap{
			"attributes": []GenericMap{
				{
					"key":    "gcp.project_id",
					"value":  config[gcpProjectIdKey],
					"action": "insert",
				},
			},
		}
		currentConfig.Processors[processorName] = processorConfig

		tracesPipelineName := "traces/googlecloudotlp-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}

		extensionConfig := GenericMap{}
		if val, exists := config[gcpProjectIdKey]; exists {
			extensionConfig["project"] = val
		}
		if val, exists := config[gcpBillingProjectIdKey]; exists {
			extensionConfig["quota_project"] = val
		}
		currentConfig.Extensions[extensionName] = extensionConfig
		currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, extensionName)
	}

	return nil, nil
}

// Implement CollectorSpecConfigurer for GoogleCloudOTLP
var _ CollectorSpecConfigurer = (*GoogleCloudOTLP)(nil)

func (g *GoogleCloudOTLP) GetCollectorSpec(dest ExporterConfigurer) *CollectorSpec {
	spec := &CollectorSpec{}
	config := dest.GetConfig()

	// If GCP_APPLICATION_CREDENTIALS is set, mount the secret and set the environment variable
	if val, exists := config[gcpApplicationCredentialsKey]; exists && val != "" && dest.GetSecretRef() != nil {
		// Add volume mount
		spec.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      strings.ReplaceAll(dest.GetSecretRef().Name, ".", "-"),
				MountPath: gcpCredentialsMountPath,
			},
		}

		// Add volume
		spec.Volumes = []corev1.Volume{
			{
				Name: strings.ReplaceAll(dest.GetSecretRef().Name, ".", "-"),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dest.GetSecretRef().Name,
						Items: []corev1.KeyToPath{
							{
								Key:  gcpApplicationCredentialsKey,
								Path: gcpApplicationCredentialsKey,
							},
						},
					},
				},
			},
		}

		// Add environment variable pointing to the mounted credentials
		spec.EnvVars = []corev1.EnvVar{
			{
				Name:  "GOOGLE_APPLICATION_CREDENTIALS",
				Value: gcpCredentialsMountPath + "/" + gcpApplicationCredentialsKey,
			},
		}
	}

	return spec
}
