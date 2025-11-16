package nodecollector

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/autoscaler/controllers/nodecollector/collectorconfig"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (b *nodeCollectorBaseReconciler) SyncConfigMap(ctx context.Context, sources *odigosv1.InstrumentationConfigList, clusterCollectorGroup odigosv1.CollectorsGroup, allProcessors *odigosv1.ProcessorList,
	datacollection *odigosv1.CollectorsGroup) error {
	logger := log.FromContext(ctx)

	processors := commonconf.FilterAndSortProcessorsByOrderHint(allProcessors, odigosv1.CollectorsGroupRoleNodeCollector)

	if b.autoscalerDeployment == nil {
		// we only need to get the autoscaler deployment once since it can't change while this code is running
		// (since we are running in the autoscaler pod)
		autoscalerDeployment := &appsv1.Deployment{}
		err := b.Client.Get(ctx, client.ObjectKey{Namespace: env.GetCurrentNamespace(), Name: k8sconsts.AutoScalerDeploymentName}, autoscalerDeployment)
		if err != nil {
			return err
		}
		b.autoscalerDeployment = autoscalerDeployment
	}

	desired, err := b.getDesiredConfigMap(ctx, sources, clusterCollectorGroup, processors, datacollection)
	if err != nil {
		logger.Error(err, "failed to get desired config map")
		return err
	}

	existing := &v1.ConfigMap{}
	if err := b.Client.Get(ctx, client.ObjectKey{Namespace: env.GetCurrentNamespace(), Name: k8sconsts.OdigosNodeCollectorConfigMapName}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("creating config map")
			_, err := b.createConfigMap(desired, ctx)
			if err != nil {
				logger.Error(err, "failed to create config map")
				return err
			}
			return nil
		} else {
			logger.Error(err, "failed to get config map")
			return err
		}
	}

	logger.V(0).Info("Patching config map")
	_, err = b.patchConfigMap(ctx, existing, desired)
	if err != nil {
		logger.Error(err, "failed to patch config map")
		return err
	}

	return nil
}

func (b *nodeCollectorBaseReconciler) patchConfigMap(ctx context.Context, existing *v1.ConfigMap, desired *v1.ConfigMap) (*v1.ConfigMap, error) {
	if reflect.DeepEqual(existing.Data, desired.Data) &&
		reflect.DeepEqual(existing.ObjectMeta.OwnerReferences, desired.ObjectMeta.OwnerReferences) {
		log.FromContext(ctx).V(0).Info("Config maps already match")
		return existing, nil
	}
	updated := existing.DeepCopy()
	updated.Data = desired.Data
	updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences
	patch := client.MergeFrom(existing)
	if err := b.Client.Patch(ctx, updated, patch); err != nil {
		return nil, err
	}

	return updated, nil
}

func (b *nodeCollectorBaseReconciler) createConfigMap(desired *v1.ConfigMap, ctx context.Context) (*v1.ConfigMap, error) {
	if err := b.Client.Create(ctx, desired); err != nil {
		return nil, err
	}

	return desired, nil
}

func noopConfigMap() (string, error) {
	config := config.Config{
		Extensions: config.GenericMap{
			"health_check": config.GenericMap{
				"endpoint": "0.0.0.0:13133",
			},
		},
		Receivers: config.GenericMap{
			"otlp": config.GenericMap{
				"protocols": config.GenericMap{
					"grpc": config.GenericMap{
						"endpoint": "0.0.0.0:4317",
					},
					"http": config.GenericMap{
						"endpoint": "0.0.0.0:4318",
					},
				},
			},
		},
		Exporters: config.GenericMap{
			"nop": config.GenericMap{},
		},
		Service: config.Service{
			Extensions: []string{"health_check"},
			Pipelines: map[string]config.Pipeline{
				"traces": {
					Receivers:  []string{"otlp"},
					Processors: []string{},
					Exporters:  []string{"nop"},
				},
				"metrics": {
					Receivers:  []string{"otlp"},
					Processors: []string{},
					Exporters:  []string{"nop"},
				},
				"logs": {
					Receivers:  []string{"otlp"},
					Processors: []string{},
					Exporters:  []string{"nop"},
				},
			},
		},
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(yamlData), nil
}

func (b *nodeCollectorBaseReconciler) getDesiredConfigMap(ctx context.Context, sources *odigosv1.InstrumentationConfigList, clusterCollectorGroup odigosv1.CollectorsGroup, processors []*odigosv1.Processor,
	cg *odigosv1.CollectorsGroup) (*v1.ConfigMap, error) {
	if b.autoscalerDeployment == nil {
		return nil, errors.New("autoscaler deployment is not set in the reconciler, cannot set owner reference")
	}
	var err error
	var cmData string

	tracingLoadBalancingNeeded, err := isTracingLoadBalancingNeeded(ctx, b.Client, clusterCollectorGroup)
	if err != nil {
		return nil, err
	}

	if cg == nil || len(clusterCollectorGroup.Status.ReceiverSignals) == 0 {
		// if collectors group is not created yet, or there are no signals to collect, return a no-op configmap
		// this can happen if no sources are instrumented yet or no destinations are added.
		cmData, err = noopConfigMap()
	} else {
		cmData, err = calculateConfigMapData(cg, sources, clusterCollectorGroup.Status.ReceiverSignals, processors, commonconf.ControllerConfig.OnGKE, tracingLoadBalancingNeeded)
	}

	if err != nil {
		return nil, err
	}

	desired := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorConfigMapName,
			Namespace: env.GetCurrentNamespace(),
		},
		Data: map[string]string{
			k8sconsts.OdigosNodeCollectorConfigMapKey: cmData,
		},
	}

	// set the autoscaler deployment as the owner of the configmap
	// since it is the one creating it and updating it.
	// cg might not yet exist and failing to have an owner will lead to un-cleaned resources on uninstall.
	if err := ctrl.SetControllerReference(b.autoscalerDeployment, &desired, b.scheme); err != nil {
		return nil, err
	}

	return &desired, nil
}

func calculateConfigMapData(
	nodeCG *odigosv1.CollectorsGroup,
	sources *odigosv1.InstrumentationConfigList,
	clusterCollectorSignals []odigoscommon.ObservabilitySignal,
	processors []*odigosv1.Processor,
	onGKE bool,
	loadBalancingNeeded bool) (string, error) {

	ownMetricsPort := nodeCG.Spec.CollectorOwnMetricsPort
	odigosNamespace := env.GetCurrentNamespace()

	processorsResults := config.CrdProcessorToConfig(commonconf.ToProcessorConfigurerArray(processors))
	for name, err := range processorsResults.Errs {
		log.Log.V(0).Error(err, "failed to convert processor manifest to config", "processor", name)
		return "", err
	}

	// common config domains - always set and active
	activeConfigDomains := []config.Config{
		collectorconfig.CommonConfig(nodeCG, onGKE),
		collectorconfig.OwnMetricsConfig(ownMetricsPort),
		processorsResults.ProcessorsConfig,
	}

	// metrics
	metricsEnabled := slices.Contains(clusterCollectorSignals, odigoscommon.MetricsObservabilitySignal)
	metricsConfigSettings := nodeCG.Spec.Metrics
	var additionalTraceExporters []string
	if metricsEnabled && metricsConfigSettings != nil {

		// span metrics
		additionalMetricsRecivers := []string{}
		if metricsConfigSettings.SpanMetrics != nil {
			spanMetricsConfig, additionalSpanMetricsTraceExporters, _ := collectorconfig.GetSpanMetricsConfig(*metricsConfigSettings.SpanMetrics)
			additionalTraceExporters = append(additionalTraceExporters, additionalSpanMetricsTraceExporters...)
			// NOTICE: temporarily bypass the normal metrics pipeline.
			// this is to allow span metrics to be reported without any additional metric resource attributes.
			// once finer control is implemented as to what resource attributes are included in the metrics pipeline,
			// we can send span metrics back into the normal metrics pipeline.
			// additionalMetricsRecivers = append(additionalMetricsRecivers, additionalSpanMetricsMetricsReceivers...)
			activeConfigDomains = append(activeConfigDomains, spanMetricsConfig)
		}

		metricsConfig := collectorconfig.MetricsConfig(nodeCG, odigosNamespace, processorsResults.MetricsProcessors, additionalMetricsRecivers, metricsConfigSettings)
		activeConfigDomains = append(activeConfigDomains, metricsConfig)
	}

	// traces
	tracesEnabledInClusterCollector := slices.Contains(clusterCollectorSignals, odigoscommon.TracesObservabilitySignal)
	// create traces pipeline if either:
	// - cluster collector has traces enabled (trace destination is enabled)
	// - there are additional trace exporters (e.g. spanmetrics connector)
	if tracesEnabledInClusterCollector || len(additionalTraceExporters) > 0 {
		tracesConfig := collectorconfig.TracesConfig(nodeCG, odigosNamespace, processorsResults.TracesProcessors, processorsResults.TracesProcessorsPostSpanMetrics, additionalTraceExporters, tracesEnabledInClusterCollector, loadBalancingNeeded)
		activeConfigDomains = append(activeConfigDomains, tracesConfig)
	}

	// logs
	collectLogs := slices.Contains(clusterCollectorSignals, odigoscommon.LogsObservabilitySignal)
	if collectLogs {
		logsConfig := collectorconfig.LogsConfig(nodeCG, odigosNamespace, processorsResults.LogsProcessors, sources)
		activeConfigDomains = append(activeConfigDomains, logsConfig)
	}

	// merge all config domains into one collector config
	mergedConfig, err := config.MergeConfigs(activeConfigDomains...)
	if err != nil {
		return "", err
	}

	data, err := yaml.Marshal(mergedConfig)
	if err != nil {
		return "", err
	}

	return string(data), nil
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
