package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	qrynOssHost                          = "QRYN_OSS_URL"
	qrynOssUsername                      = "QRYN_OSS_USERNAME"
	qrynOssresourceToTelemetryConversion = "QRYN_OSS_RESOURCE_TO_TELEMETRY_CONVERSION"
	qrynOssAddExporterName               = "QRYN_OSS_ADD_EXPORTER_NAME"
)

type QrynOSS struct {
	*Qryn
}

type QrynOssDest struct {
	ExporterConfigurer
}

func (d QrynOssDest) GetConfig() map[string]string {
	conf := d.ExporterConfigurer.GetConfig()
	conf[qrynHost] = conf[qrynOssHost]
	conf[qrynAPIKey] = conf[qrynOssUsername]
	conf[resourceToTelemetryConversion] = conf[qrynOssresourceToTelemetryConversion]
	conf[qrynSecretsOptional] = "1"
	conf[qrynAddExporterName] = conf[qrynOssAddExporterName]
	conf[qrynPasswordFieldName] = "QRYN_OSS_PASSWORD"
	return conf
}

func (g *QrynOSS) DestType() common.DestinationType {
	return common.QrynOSSDestinationType
}

func (g *QrynOSS) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	return g.Qryn.ModifyConfig(QrynOssDest{dest}, currentConfig)
}
