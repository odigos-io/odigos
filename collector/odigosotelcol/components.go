// Code generated by "go.opentelemetry.io/collector/cmd/builder". DO NOT EDIT.

package main

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	forwardconnector "go.opentelemetry.io/collector/connector/forwardconnector"
	countconnector "github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector"
	datadogconnector "github.com/open-telemetry/opentelemetry-collector-contrib/connector/datadogconnector"
	exceptionsconnector "github.com/open-telemetry/opentelemetry-collector-contrib/connector/exceptionsconnector"
	routingconnector "github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector"
	servicegraphconnector "github.com/open-telemetry/opentelemetry-collector-contrib/connector/servicegraphconnector"
	spanmetricsconnector "github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector"
	debugexporter "go.opentelemetry.io/collector/exporter/debugexporter"
	nopexporter "go.opentelemetry.io/collector/exporter/nopexporter"
	otlpexporter "go.opentelemetry.io/collector/exporter/otlpexporter"
	otlphttpexporter "go.opentelemetry.io/collector/exporter/otlphttpexporter"
	azureblobstorageexporter "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/azureblobstorageexporter"
	googlecloudstorageexporter "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/googlecloudstorageexporter"
	mockdestinationexporter "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/mockdestinationexporter"
	awscloudwatchlogsexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awscloudwatchlogsexporter"
	awss3exporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awss3exporter"
	awsxrayexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsxrayexporter"
	azuredataexplorerexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuredataexplorerexporter"
	azuremonitorexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuremonitorexporter"
	carbonexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/carbonexporter"
	clickhouseexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter"
	cassandraexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/cassandraexporter"
	coralogixexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/coralogixexporter"
	datadogexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter"
	datasetexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datasetexporter"
	elasticsearchexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	fileexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/fileexporter"
	googlecloudexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter"
	googlecloudpubsubexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudpubsubexporter"
	googlemanagedprometheusexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlemanagedprometheusexporter"
	honeycombmarkerexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/honeycombmarkerexporter"
	influxdbexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/influxdbexporter"
	kafkaexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"
	loadbalancingexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/loadbalancingexporter"
	logicmonitorexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logicmonitorexporter"
	logzioexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logzioexporter"
	lokiexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/lokiexporter"
	mezmoexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/mezmoexporter"
	opencensusexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opencensusexporter"
	opensearchexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opensearchexporter"
	prometheusexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter"
	prometheusremotewriteexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter"
	pulsarexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/pulsarexporter"
	sapmexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sapmexporter"
	sentryexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sentryexporter"
	signalfxexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/signalfxexporter"
	splunkhecexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/splunkhecexporter"
	sumologicexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sumologicexporter"
	syslogexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/syslogexporter"
	tencentcloudlogserviceexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/tencentcloudlogserviceexporter"
	zipkinexporter "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter"
	zpagesextension "go.opentelemetry.io/collector/extension/zpagesextension"
	healthcheckextension "github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension"
	pprofextension "github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension"
	basicauthextension "github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension"
	odigosresourcenameprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigosresourcenameprocessor"
	odigossamplingprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor"
	odigosconditionalattributes "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigosconditionalattributes"
	odigossqldboperationprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossqldboperationprocessor"
	batchprocessor "go.opentelemetry.io/collector/processor/batchprocessor"
	memorylimiterprocessor "go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	attributesprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	cumulativetodeltaprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/cumulativetodeltaprocessor"
	deltatorateprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatorateprocessor"
	filterprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	groupbyattrsprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbyattrsprocessor"
	groupbytraceprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor"
	k8sattributesprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor"
	metricsgenerationprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricsgenerationprocessor"
	metricstransformprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor"
	probabilisticsamplerprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor"
	redactionprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor"
	resourcedetectionprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor"
	resourceprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor"
	routingprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/routingprocessor"
	sumologicprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/sumologicprocessor"
	spanprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanprocessor"
	tailsamplingprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor"
	transformprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	remotetapprocessor "github.com/open-telemetry/opentelemetry-collector-contrib/processor/remotetapprocessor"
	odigostrafficmetrics "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigostrafficmetrics"
	odigossourcesfilterprocessor "github.com/odigos-io/odigos/processor/odigossourcesfilterprocessor"
	otlpreceiver "go.opentelemetry.io/collector/receiver/otlpreceiver"
	zipkinreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver"
	filelogreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver"
	kubeletstatsreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kubeletstatsreceiver"
	hostmetricsreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"
	prometheusreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver"
)

func components() (otelcol.Factories, error) {
	var err error
	factories := otelcol.Factories{}

	factories.Extensions, err = extension.MakeFactoryMap(
		zpagesextension.NewFactory(),
		healthcheckextension.NewFactory(),
		pprofextension.NewFactory(),
		basicauthextension.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}
	factories.ExtensionModules = make(map[component.Type]string, len(factories.Extensions))
	factories.ExtensionModules[zpagesextension.NewFactory().Type()] = "go.opentelemetry.io/collector/extension/zpagesextension v0.119.0"
	factories.ExtensionModules[healthcheckextension.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension v0.119.0"
	factories.ExtensionModules[pprofextension.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension v0.119.0"
	factories.ExtensionModules[basicauthextension.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension v0.119.0"

	factories.Receivers, err = receiver.MakeFactoryMap(
		otlpreceiver.NewFactory(),
		zipkinreceiver.NewFactory(),
		filelogreceiver.NewFactory(),
		kubeletstatsreceiver.NewFactory(),
		hostmetricsreceiver.NewFactory(),
		prometheusreceiver.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}
	factories.ReceiverModules = make(map[component.Type]string, len(factories.Receivers))
	factories.ReceiverModules[otlpreceiver.NewFactory().Type()] = "go.opentelemetry.io/collector/receiver/otlpreceiver v0.119.0"
	factories.ReceiverModules[zipkinreceiver.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver v0.119.0"
	factories.ReceiverModules[filelogreceiver.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver v0.119.0"
	factories.ReceiverModules[kubeletstatsreceiver.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kubeletstatsreceiver v0.119.0"
	factories.ReceiverModules[hostmetricsreceiver.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver v0.119.0"
	factories.ReceiverModules[prometheusreceiver.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver v0.119.0"

	factories.Exporters, err = exporter.MakeFactoryMap(
		debugexporter.NewFactory(),
		nopexporter.NewFactory(),
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
		azureblobstorageexporter.NewFactory(),
		googlecloudstorageexporter.NewFactory(),
		mockdestinationexporter.NewFactory(),
		awscloudwatchlogsexporter.NewFactory(),
		awss3exporter.NewFactory(),
		awsxrayexporter.NewFactory(),
		azuredataexplorerexporter.NewFactory(),
		azuremonitorexporter.NewFactory(),
		carbonexporter.NewFactory(),
		clickhouseexporter.NewFactory(),
		cassandraexporter.NewFactory(),
		coralogixexporter.NewFactory(),
		datadogexporter.NewFactory(),
		datasetexporter.NewFactory(),
		elasticsearchexporter.NewFactory(),
		fileexporter.NewFactory(),
		googlecloudexporter.NewFactory(),
		googlecloudpubsubexporter.NewFactory(),
		googlemanagedprometheusexporter.NewFactory(),
		honeycombmarkerexporter.NewFactory(),
		influxdbexporter.NewFactory(),
		kafkaexporter.NewFactory(),
		loadbalancingexporter.NewFactory(),
		logicmonitorexporter.NewFactory(),
		logzioexporter.NewFactory(),
		lokiexporter.NewFactory(),
		mezmoexporter.NewFactory(),
		opencensusexporter.NewFactory(),
		opensearchexporter.NewFactory(),
		prometheusexporter.NewFactory(),
		prometheusremotewriteexporter.NewFactory(),
		pulsarexporter.NewFactory(),
		sapmexporter.NewFactory(),
		sentryexporter.NewFactory(),
		signalfxexporter.NewFactory(),
		splunkhecexporter.NewFactory(),
		sumologicexporter.NewFactory(),
		syslogexporter.NewFactory(),
		tencentcloudlogserviceexporter.NewFactory(),
		zipkinexporter.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}
	factories.ExporterModules = make(map[component.Type]string, len(factories.Exporters))
	factories.ExporterModules[debugexporter.NewFactory().Type()] = "go.opentelemetry.io/collector/exporter/debugexporter v0.119.0"
	factories.ExporterModules[nopexporter.NewFactory().Type()] = "go.opentelemetry.io/collector/exporter/nopexporter v0.119.0"
	factories.ExporterModules[otlpexporter.NewFactory().Type()] = "go.opentelemetry.io/collector/exporter/otlpexporter v0.119.0"
	factories.ExporterModules[otlphttpexporter.NewFactory().Type()] = "go.opentelemetry.io/collector/exporter/otlphttpexporter v0.119.0"
	factories.ExporterModules[azureblobstorageexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/azureblobstorageexporter v0.119.0"
	factories.ExporterModules[googlecloudstorageexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/googlecloudstorageexporter v0.119.0"
	factories.ExporterModules[mockdestinationexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/mockdestinationexporter v0.119.0"
	factories.ExporterModules[awscloudwatchlogsexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awscloudwatchlogsexporter v0.119.0"
	factories.ExporterModules[awss3exporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awss3exporter v0.119.0"
	factories.ExporterModules[awsxrayexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsxrayexporter v0.119.0"
	factories.ExporterModules[azuredataexplorerexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuredataexplorerexporter v0.119.0"
	factories.ExporterModules[azuremonitorexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuremonitorexporter v0.119.0"
	factories.ExporterModules[carbonexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/carbonexporter v0.119.0"
	factories.ExporterModules[clickhouseexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter v0.119.0"
	factories.ExporterModules[cassandraexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/cassandraexporter v0.119.0"
	factories.ExporterModules[coralogixexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/coralogixexporter v0.119.0"
	factories.ExporterModules[datadogexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter v0.119.0"
	factories.ExporterModules[datasetexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datasetexporter v0.119.0"
	factories.ExporterModules[elasticsearchexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter v0.119.0"
	factories.ExporterModules[fileexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/fileexporter v0.119.0"
	factories.ExporterModules[googlecloudexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter v0.119.0"
	factories.ExporterModules[googlecloudpubsubexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudpubsubexporter v0.119.0"
	factories.ExporterModules[googlemanagedprometheusexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlemanagedprometheusexporter v0.119.0"
	factories.ExporterModules[honeycombmarkerexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/honeycombmarkerexporter v0.119.0"
	factories.ExporterModules[influxdbexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/influxdbexporter v0.119.0"
	factories.ExporterModules[kafkaexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter v0.119.0"
	factories.ExporterModules[loadbalancingexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/loadbalancingexporter v0.119.0"
	factories.ExporterModules[logicmonitorexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logicmonitorexporter v0.119.0"
	factories.ExporterModules[logzioexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logzioexporter v0.119.0"
	factories.ExporterModules[lokiexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/lokiexporter v0.119.0"
	factories.ExporterModules[mezmoexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/mezmoexporter v0.119.0"
	factories.ExporterModules[opencensusexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opencensusexporter v0.119.0"
	factories.ExporterModules[opensearchexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opensearchexporter v0.119.0"
	factories.ExporterModules[prometheusexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter v0.119.0"
	factories.ExporterModules[prometheusremotewriteexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter v0.119.0"
	factories.ExporterModules[pulsarexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/pulsarexporter v0.119.0"
	factories.ExporterModules[sapmexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sapmexporter v0.119.0"
	factories.ExporterModules[sentryexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sentryexporter v0.119.0"
	factories.ExporterModules[signalfxexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/signalfxexporter v0.119.0"
	factories.ExporterModules[splunkhecexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/splunkhecexporter v0.119.0"
	factories.ExporterModules[sumologicexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sumologicexporter v0.119.0"
	factories.ExporterModules[syslogexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/syslogexporter v0.119.0"
	factories.ExporterModules[tencentcloudlogserviceexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/tencentcloudlogserviceexporter v0.119.0"
	factories.ExporterModules[zipkinexporter.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter v0.119.0"

	factories.Processors, err = processor.MakeFactoryMap(
		odigosresourcenameprocessor.NewFactory(),
		odigossamplingprocessor.NewFactory(),
		odigosconditionalattributes.NewFactory(),
		odigossqldboperationprocessor.NewFactory(),
		batchprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
		attributesprocessor.NewFactory(),
		cumulativetodeltaprocessor.NewFactory(),
		deltatorateprocessor.NewFactory(),
		filterprocessor.NewFactory(),
		groupbyattrsprocessor.NewFactory(),
		groupbytraceprocessor.NewFactory(),
		k8sattributesprocessor.NewFactory(),
		metricsgenerationprocessor.NewFactory(),
		metricstransformprocessor.NewFactory(),
		probabilisticsamplerprocessor.NewFactory(),
		redactionprocessor.NewFactory(),
		resourcedetectionprocessor.NewFactory(),
		resourceprocessor.NewFactory(),
		routingprocessor.NewFactory(),
		sumologicprocessor.NewFactory(),
		spanprocessor.NewFactory(),
		tailsamplingprocessor.NewFactory(),
		transformprocessor.NewFactory(),
		remotetapprocessor.NewFactory(),
		odigostrafficmetrics.NewFactory(),
		odigossourcesfilterprocessor.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}
	factories.ProcessorModules = make(map[component.Type]string, len(factories.Processors))
	factories.ProcessorModules[odigosresourcenameprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigosresourcenameprocessor v0.119.0"
	factories.ProcessorModules[odigossamplingprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor v0.119.0"
	factories.ProcessorModules[odigosconditionalattributes.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigosconditionalattributes v0.119.0"
	factories.ProcessorModules[odigossqldboperationprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossqldboperationprocessor v0.119.0"
	factories.ProcessorModules[batchprocessor.NewFactory().Type()] = "go.opentelemetry.io/collector/processor/batchprocessor v0.119.0"
	factories.ProcessorModules[memorylimiterprocessor.NewFactory().Type()] = "go.opentelemetry.io/collector/processor/memorylimiterprocessor v0.119.0"
	factories.ProcessorModules[attributesprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor v0.119.0"
	factories.ProcessorModules[cumulativetodeltaprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/cumulativetodeltaprocessor v0.119.0"
	factories.ProcessorModules[deltatorateprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatorateprocessor v0.119.0"
	factories.ProcessorModules[filterprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor v0.119.0"
	factories.ProcessorModules[groupbyattrsprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbyattrsprocessor v0.119.0"
	factories.ProcessorModules[groupbytraceprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor v0.119.0"
	factories.ProcessorModules[k8sattributesprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor v0.119.0"
	factories.ProcessorModules[metricsgenerationprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricsgenerationprocessor v0.119.0"
	factories.ProcessorModules[metricstransformprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor v0.119.0"
	factories.ProcessorModules[probabilisticsamplerprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor v0.119.0"
	factories.ProcessorModules[redactionprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor v0.119.0"
	factories.ProcessorModules[resourcedetectionprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor v0.119.0"
	factories.ProcessorModules[resourceprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor v0.119.0"
	factories.ProcessorModules[routingprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/routingprocessor v0.119.0"
	factories.ProcessorModules[sumologicprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/sumologicprocessor v0.119.0"
	factories.ProcessorModules[spanprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanprocessor v0.119.0"
	factories.ProcessorModules[tailsamplingprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor v0.119.0"
	factories.ProcessorModules[transformprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor v0.119.0"
	factories.ProcessorModules[remotetapprocessor.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/remotetapprocessor v0.119.0"
	factories.ProcessorModules[odigostrafficmetrics.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigostrafficmetrics v0.119.0"
	factories.ProcessorModules[odigossourcesfilterprocessor.NewFactory().Type()] = "github.com/odigos-io/odigos/processor/odigossourcesfilterprocessor v0.119.0"

	factories.Connectors, err = connector.MakeFactoryMap(
		forwardconnector.NewFactory(),
		countconnector.NewFactory(),
		datadogconnector.NewFactory(),
		exceptionsconnector.NewFactory(),
		routingconnector.NewFactory(),
		servicegraphconnector.NewFactory(),
		spanmetricsconnector.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, err
	}
	factories.ConnectorModules = make(map[component.Type]string, len(factories.Connectors))
	factories.ConnectorModules[forwardconnector.NewFactory().Type()] = "go.opentelemetry.io/collector/connector/forwardconnector v0.119.0"
	factories.ConnectorModules[countconnector.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector v0.119.0"
	factories.ConnectorModules[datadogconnector.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/connector/datadogconnector v0.119.0"
	factories.ConnectorModules[exceptionsconnector.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/connector/exceptionsconnector v0.119.0"
	factories.ConnectorModules[routingconnector.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector v0.119.0"
	factories.ConnectorModules[servicegraphconnector.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/connector/servicegraphconnector v0.119.0"
	factories.ConnectorModules[spanmetricsconnector.NewFactory().Type()] = "github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector v0.119.0"

	return factories, nil
}
