package nodecollector

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/autoscaler/controllers/nodecollector/collectorconfig"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

func (b *nodeCollectorBaseReconciler) SyncConfigMap(ctx context.Context, sources *odigosv1.InstrumentationConfigList, clusterCollectorGroup odigosv1.CollectorsGroup, allProcessors *odigosv1.ProcessorList,
	datacollection *odigosv1.CollectorsGroup) error {

	processors := commonconf.FilterAndSortProcessorsByOrderHint(allProcessors, odigosv1.CollectorsGroupRoleNodeCollector)

	if b.autoscalerDeployment == nil {
		// we only need to get the autoscaler deployment once since it can't change while this code is running
		// (since we are running in the autoscaler pod)
		autoscalerDeployment := &appsv1.Deployment{}
		autoscalerDeploymentName := env.GetComponentDeploymentNameOrDefault(k8sconsts.AutoScalerDeploymentName)
		err := b.Client.Get(ctx, client.ObjectKey{Namespace: b.odigosNamespace, Name: autoscalerDeploymentName}, autoscalerDeployment)
		if err != nil {
			return err
		}
		b.autoscalerDeployment = autoscalerDeployment
	}

	tracingLoadBalancingNeeded, err := isTracingLoadBalancingNeeded(ctx, b.Client, clusterCollectorGroup)
	if err != nil {
		return errors.Join(err, errors.New("failed to check if tracing load balancing is needed"))
	}

	configDomains, configAsYamlText, err := calculateCollectorConfigDomains(b.odigosNamespace, datacollection, sources, clusterCollectorGroup.Status.ReceiverSignals, processors, commonconf.ControllerConfig.OnGKE, tracingLoadBalancingNeeded)
	if err != nil {
		return errors.Join(err, errors.New("failed to calculate collector config domains"))
	}

	err = b.persistCollectorConfig(ctx, configAsYamlText)
	if err != nil {
		return errors.Join(err, errors.New("failed to persist node collector config"))
	}

	err = b.persistCollectorConfigDomains(ctx, configDomains)
	if err != nil {
		return errors.Join(err, errors.New("failed to persist node collector config domains"))
	}

	return nil
}

func (b *nodeCollectorBaseReconciler) persistCollectorConfig(ctx context.Context, configAsYamlText string) error {

	nodeCollectorCg := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorConfigMapName,
			Namespace: b.odigosNamespace,
		},
		Data: map[string]string{
			k8sconsts.OdigosNodeCollectorConfigMapKey: configAsYamlText,
		},
	}

	// set the autoscaler deployment as the owner of the configmap
	// since it is the one creating it and updating it.
	// cg might not yet exist and failing to have an owner will lead to un-cleaned resources on uninstall.
	if err := ctrl.SetControllerReference(b.autoscalerDeployment, &nodeCollectorCg, b.scheme); err != nil {
		return errors.Join(err, errors.New("failed to set owner reference to node collector config map"))
	}

	// apply the config map (override it regardless of the existing data so it will always have the latest data)
	if err := b.Client.Patch(ctx, &nodeCollectorCg, client.Apply, client.ForceOwnership, client.FieldOwner("autoscaler")); err != nil {
		return errors.Join(err, errors.New("failed to apply node collector config map in kubernetes"))
	}
	return nil
}

func (b *nodeCollectorBaseReconciler) persistCollectorConfigDomains(ctx context.Context, configDomains map[string]config.Config) error {

	data := map[string]string{}
	for domain, config := range configDomains {
		configYaml, err := yaml.Marshal(config)
		if err != nil {
			return errors.Join(err, errors.New("failed to marshal collector config domain to yaml"))
		}
		data[domain] = string(configYaml)
	}

	cmDomains := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorConfigMapConfigDomainsName,
			Namespace: b.odigosNamespace,
		},
		Data: data,
	}

	if err := ctrl.SetControllerReference(b.autoscalerDeployment, &cmDomains, b.scheme); err != nil {
		return errors.Join(err, errors.New("failed to set owner reference to node collector config map domains"))
	}

	// apply the config map (override it regardless of the existing data so it will always have the latest data)
	if err := b.Client.Patch(ctx, &cmDomains, client.Apply, client.ForceOwnership, client.FieldOwner("autoscaler")); err != nil {
		return errors.Join(err, errors.New("failed to apply node collector config map domains in kubernetes"))
	}
	return nil
}

func calculateCollectorConfigDomains(
	odigosNamespace string,
	nodeCG *odigosv1.CollectorsGroup,
	sources *odigosv1.InstrumentationConfigList,
	clusterCollectorSignals []odigoscommon.ObservabilitySignal,
	processors []*odigosv1.Processor,
	onGKE bool,
	loadBalancingNeeded bool) (map[string]config.Config, string, error) {

	// common config domains - always set and active
	configDomains := map[string]config.Config{
		"common": collectorconfig.CommonConfig(),
	}

	ownMetricsPort := k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault
	configDomains["own_metrics_ui"] = collectorconfig.OwnMetricsConfigUi(ownMetricsPort)

	// all the rest of the config is only evaluated if the node collector group is not nil
	// node collector group is nil before any sources are added in odigos or cluster collector is not yet ready.
	// this logic should be revisited in the future, but kept as is for now (nov 2025)
	if nodeCG == nil {
		mergedConfig, err := config.MergeConfigs(configDomains)
		if err != nil {
			return nil, "", errors.Join(err, errors.New("failed to merge collector config domains"))
		}
		mergedConfigYaml, err := yaml.Marshal(mergedConfig)
		if err != nil {
			return nil, "", errors.Join(err, errors.New("failed to marshal merged config to yaml"))
		}
		return configDomains, string(mergedConfigYaml), nil
	}

	// processors from k8s "Processor" custom resource
	processorsResults := config.CrdProcessorToConfig(commonconf.ToProcessorConfigurerArray(processors))
	for name, err := range processorsResults.Errs {
		log.Log.V(0).Error(err, "failed to convert processor manifest to config", "processor", name)
		return nil, "", err
	}
	configDomains["processors"] = processorsResults.ProcessorsConfig

	configDomains["common_application_telemetry"] = collectorconfig.CommonApplicationTelemetryConfig(nodeCG, onGKE, odigosNamespace)

	// metrics
	metricsEnabled := slices.Contains(clusterCollectorSignals, odigoscommon.MetricsObservabilitySignal)
	metricsConfigSettings := nodeCG.Spec.Metrics
	var additionalTraceExporters []string
	if metricsEnabled && metricsConfigSettings != nil {

		// span metrics
		if metricsConfigSettings.SpanMetrics != nil {
			spanMetricsConfig, additionalSpanMetricsTraceExporters, _ := collectorconfig.GetSpanMetricsConfig(*metricsConfigSettings.SpanMetrics)
			additionalTraceExporters = append(additionalTraceExporters, additionalSpanMetricsTraceExporters...)
			// NOTICE: temporarily bypass the normal metrics pipeline.
			// this is to allow span metrics to be reported without any additional metric resource attributes.
			// once finer control is implemented as to what resource attributes are included in the metrics pipeline,
			// we can send span metrics back into the normal metrics pipeline.
			// additionalMetricsReceivers = append(additionalMetricsReceivers, additionalSpanMetricsMetricsReceivers...)
			configDomains["span_metrics"] = spanMetricsConfig
		}

		metricsConfig := collectorconfig.MetricsConfig(nodeCG, odigosNamespace, processorsResults.MetricsProcessors, metricsConfigSettings)
		configDomains["metrics"] = metricsConfig
	}

	// ownmetrics - report the node collector's own telemetry to the cluster collector
	if nodeCG.Spec.Metrics != nil && nodeCG.Spec.Metrics.OdigosOwnMetrics != nil {
		ownMetricsConfig, err := ownMetricsTelemetryConfig(nodeCG.Spec.Metrics.OdigosOwnMetrics, odigosNamespace)
		if err != nil {
			return nil, "", errors.Join(err, errors.New("failed to calculate own metrics config"))
		}
		configDomains["own_metrics"] = ownMetricsConfig
	}

	// traces
	tracesEnabledInClusterCollector := slices.Contains(clusterCollectorSignals, odigoscommon.TracesObservabilitySignal)
	// create traces pipeline if either:
	// - cluster collector has traces enabled (trace destination is enabled)
	// - there are additional trace exporters (e.g. spanmetrics connector)
	if tracesEnabledInClusterCollector || len(additionalTraceExporters) > 0 {
		tracesConfig := collectorconfig.TracesConfig(nodeCG, odigosNamespace, processorsResults.TracesProcessors, processorsResults.TracesProcessorsPostSpanMetrics, additionalTraceExporters, tracesEnabledInClusterCollector, loadBalancingNeeded)
		configDomains["traces"] = tracesConfig
	}

	// logs
	collectLogs := slices.Contains(clusterCollectorSignals, odigoscommon.LogsObservabilitySignal)
	if collectLogs {
		logsConfig := collectorconfig.LogsConfig(nodeCG, odigosNamespace, processorsResults.LogsProcessors, sources)
		configDomains["logs"] = logsConfig
	}

	mergedConfig, err := config.MergeConfigs(configDomains)
	if err != nil {
		return nil, "", errors.Join(err, errors.New("failed to merge collector config domains"))
	}
	mergedConfigYaml, err := yaml.Marshal(mergedConfig)
	if err != nil {
		return nil, "", errors.Join(err, errors.New("failed to marshal merged config to yaml"))
	}

	return configDomains, string(mergedConfigYaml), nil
}

func ownMetricsTelemetryConfig(ownMetricsConfig *odigosv1.OdigosOwnMetricsSettings, odigosNamespace string) (config.Config, error) {
	duration, err := time.ParseDuration(ownMetricsConfig.Interval)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to parse own metrics interval %q: %w", ownMetricsConfig.Interval, err)
	}

	clusterCollectorEndpoint := fmt.Sprintf("%s.%s:44318", k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace)

	reader := config.GenericMap{
		"periodic": config.GenericMap{
			"interval": int64(duration.Milliseconds()),
			"exporter": config.GenericMap{
				"otlp": config.GenericMap{
					"endpoint": clusterCollectorEndpoint,
					"insecure": true,
					"protocol": "http/protobuf",
				},
			},
		},
	}

	return config.Config{
		Service: config.Service{
			Telemetry: config.Telemetry{
				Metrics: config.MetricsConfig{
					Readers: []config.GenericMap{reader},
				},
			},
		},
	}, nil
}

func getConfigMap(ctx context.Context, c client.Client, namespace string) (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: k8sconsts.OdigosNodeCollectorConfigMapName}, configMap); err != nil {
		return nil, err
	}

	return configMap, nil
}

func getSignalsFromOtelcolConfig(otelcolConfigContent string) ([]odigoscommon.ObservabilitySignal, error) {
	config := config.Config{}
	err := yaml.Unmarshal([]byte(otelcolConfigContent), &config)
	if err != nil {
		return nil, err
	}

	tracesEnabled := false
	metricsEnabled := false
	logsEnabled := false
	for pipelineName, pipeline := range config.Service.Pipelines {
		// only consider pipelines with `otlp` receiver
		// which are the ones that can actually receive data
		if !slices.Contains(pipeline.Receivers, collectorconfig.OTLPInReceiverName) {
			continue
		}
		if strings.HasPrefix(pipelineName, "traces") {
			tracesEnabled = true
		} else if strings.HasPrefix(pipelineName, "metrics") {
			metricsEnabled = true
		} else if strings.HasPrefix(pipelineName, "logs") {
			logsEnabled = true
		}
	}

	signals := []odigoscommon.ObservabilitySignal{}
	if tracesEnabled {
		signals = append(signals, odigoscommon.TracesObservabilitySignal)
	}
	if metricsEnabled {
		signals = append(signals, odigoscommon.MetricsObservabilitySignal)
	}
	if logsEnabled {
		signals = append(signals, odigoscommon.LogsObservabilitySignal)
	}

	return signals, nil
}

func isSamplingActionsEnabled(actions *odigosv1.ActionList) bool {
	for _, action := range actions.Items {
		// If the action is disabled, skip it
		if action.Spec.Disabled {
			continue
		}
		// Only sampling actions that are not disabled should be considered
		if action.Spec.Samplers != nil {
			return true
		}
	}
	return false
}

func isTracingLoadBalancingNeeded(ctx context.Context, client client.Client, clusterCollectorGroup odigosv1.CollectorsGroup) (bool, error) {
	// We're enabling load balancing for traces only if one of the following conditions is met:
	// 1. Sampling actions are enabled
	// 2. Service graph is enabled
	serviceGraphEnabled := clusterCollectorGroup.Spec.ServiceGraphDisabled == nil || !*clusterCollectorGroup.Spec.ServiceGraphDisabled

	if serviceGraphEnabled {
		return true, nil
	}

	actions := odigosv1.ActionList{}
	if err := client.List(ctx, &actions); err != nil {
		return false, err
	}
	samplingActionsEnabled := isSamplingActionsEnabled(&actions)

	return samplingActionsEnabled, nil
}
