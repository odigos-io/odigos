package config

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

var availableConfigers = []Configer{
	&AlibabaCloud{},
	&AppDynamics{},
	&AWSCloudWatch{},
	&AWSS3{},
	&AWSXRay{},
	&Axiom{},
	&AzureBlobStorage{},
	&BetterStack{},
	&Bonree{},
	&Causely{},
	&Checkly{},
	&Chronosphere{},
	&Clickhouse{},
	&Coralogix{},
	&Dash0{},
	&Datadog{},
	&Debug{},
	&Dynamic{},
	&Dynatrace{},
	&ElasticAPM{},
	&Elasticsearch{},
	&GenericOTLP{},
	&GoogleCloud{},
	&GoogleCloudStorage{},
	&GrafanaCloudLoki{},
	&GrafanaCloudPrometheus{},
	&GrafanaCloudTempo{},
	&Greptime{},
	&Groundcover{},
	&Honeycomb{},
	&HyperDX{},
	&Instana{},
	&Jaeger{},
	&Kafka{},
	&KloudMate{},
	&Last9{},
	&Lightstep{},
	&Logzio{},
	&Loki{},
	&Lumigo{},
	&Middleware{},
	&Mock{},
	&NewRelic{},
	&Nop{},
	&Observe{},
	&OneUptime{},
	&OpenObserve{},
	&OpsVerse{},
	&Oracle{},
	&OTLPHttp{},
	&Prometheus{},
	&Qryn{},
	&QrynOSS{},
	&Quickwit{},
	&Sentry{},
	&Seq{},
	&Signoz{},
	&Splunk{},
	&SumoLogic{},
	&TelemetryHub{},
	&Tempo{},
	&Tingyun{},
	&Traceloop{},
	&Uptrace{},
	&VictoriaMetricsCloud{},
}

type Configer interface {
	DestType() common.DestinationType
	ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error)
}

type ResourceStatuses struct {
	Destination map[string]error
	Processor   map[string]error
}

func LoadConfigers() (map[common.DestinationType]Configer, error) {
	configers := map[common.DestinationType]Configer{}
	for _, configer := range availableConfigers {
		if _, exists := configers[configer.DestType()]; exists {
			return nil, fmt.Errorf("duplicate configer for %s", configer.DestType())
		}

		configers[configer.DestType()] = configer
	}
	fmt.Println("configers", configers)
	return configers, nil
}

func isSignalExists(dest SignalSpecific, signal common.ObservabilitySignal) bool {
	for _, s := range dest.GetSignals() {
		if s == signal {
			return true
		}
	}

	return false
}

func IsTracingEnabled(dest SignalSpecific) bool {
	return isSignalExists(dest, common.TracesObservabilitySignal)
}

func IsMetricsEnabled(dest SignalSpecific) bool {
	return isSignalExists(dest, common.MetricsObservabilitySignal)
}

func IsLoggingEnabled(dest SignalSpecific) bool {
	return isSignalExists(dest, common.LogsObservabilitySignal)
}

func addProtocol(s string) string {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}

	return fmt.Sprintf("http://%s", s)
}
