package nodecollectorsgroup

import (
	"context"
	"errors"
	"slices"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/scheduler/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// the default memory request in MiB
	defaultRequestMemoryMiB = 256

	// this configures the processor limit_mib, which is the hard limit in MiB, afterwhich garbage collection will be forced.
	// as recommended by the processor docs, if not set, this is set to 50MiB less than the memory limit of the collector
	defaultMemoryLimiterLimitDiffMib = 50

	// the soft limit will be set to 80% of the hard limit.
	// this value is used to derive the "spike_limit_mib" parameter in the processor configuration if a value is not set
	defaultMemoryLimiterSpikePercentage = 20.0

	// the percentage out of the memory limiter hard limit, at which go runtime will start garbage collection.
	// it is used to calculate the GOMEMLIMIT environment variable value.
	defaultGoMemLimitPercentage = 80.0

	// the memory settings should prevent the collector from exceeding the memory request.
	// however, the mechanism is heuristic and does not guarantee to prevent OOMs.
	// allowing the memory limit to be slightly above the memory request can help in reducing the chances of OOMs in edge cases.
	// instead of having the process killed, it can use extra memory available on the node without allocating it preemptively.
	memoryLimitAboveRequestFactor = 2.0

	// the default CPU request in millicores
	defaultRequestCPUm = 250
	// the default CPU limit in millicores
	defaultLimitCPUm = 500
)

func getResourceSettings(odigosConfiguration common.OdigosConfiguration) odigosv1.CollectorsGroupResourcesSettings {
	// memory request is expensive on daemonsets since it will consume this memory
	// on each node in the cluster. setting to 256, but allowing memory to spike higher
	// to consume more available memory on the node.
	// if the node has memory to spare, we can use it to buffer more data before dropping,
	// but it also means that if no memory is available, collector might get killed by OOM killer.
	//
	// we can trade-off the memory request:
	// - more memory request: more memory allocated per collector on each node, but more buffer for bursts and transient failures.
	// - less memory request: efficient use of cluster resources, but data might be dropped earlier on spikes.
	// currently choosing 256MiB as a balance (~200MiB left for heap to handle batches and export queues).
	//
	// we can trade-off how high the memory limit is set above the request:
	// - limit is set to request: collector most stable (no OOM) but smaller buffer for bursts and early data drop.
	// - limit is set way above request: in case of memory spike, collector will use extra memory available on the node to buffer data, but might get killed by OOM killer if this memory is not available.
	// currently choosing 512MiB as a balance (200MiB guaranteed for heap, and the rest ~300MiB of buffer from node before start dropping).

	nodeCollectorConfig := odigosConfiguration.CollectorNode

	memoryRequestMiB := defaultRequestMemoryMiB
	if nodeCollectorConfig != nil && nodeCollectorConfig.RequestMemoryMiB > 0 {
		memoryRequestMiB = nodeCollectorConfig.RequestMemoryMiB
	}
	memoryLimitMiB := int(float64(memoryRequestMiB) * memoryLimitAboveRequestFactor)
	if nodeCollectorConfig != nil && nodeCollectorConfig.LimitMemoryMiB > 0 {
		memoryLimitMiB = nodeCollectorConfig.LimitMemoryMiB
	}

	memoryLimiterLimitMiB := memoryLimitMiB - defaultMemoryLimiterLimitDiffMib
	if nodeCollectorConfig != nil && nodeCollectorConfig.MemoryLimiterLimitMiB > 0 {
		memoryLimiterLimitMiB = nodeCollectorConfig.MemoryLimiterLimitMiB
	}
	memoryLimiterSpikeLimitMiB := memoryLimiterLimitMiB * defaultMemoryLimiterSpikePercentage / 100
	if nodeCollectorConfig != nil && nodeCollectorConfig.MemoryLimiterSpikeLimitMiB > 0 {
		memoryLimiterSpikeLimitMiB = nodeCollectorConfig.MemoryLimiterSpikeLimitMiB
	}

	gomemlimitMiB := int(memoryLimiterLimitMiB * defaultGoMemLimitPercentage / 100.0)
	if nodeCollectorConfig != nil && nodeCollectorConfig.GoMemLimitMib != 0 {
		gomemlimitMiB = nodeCollectorConfig.GoMemLimitMib
	}

	cpuRequestm := defaultRequestCPUm
	if nodeCollectorConfig != nil && nodeCollectorConfig.RequestCPUm > 0 {
		cpuRequestm = nodeCollectorConfig.RequestCPUm
	}
	cpuLimitm := defaultLimitCPUm
	if nodeCollectorConfig != nil && nodeCollectorConfig.LimitCPUm > 0 {
		cpuLimitm = nodeCollectorConfig.LimitCPUm
	}

	return odigosv1.CollectorsGroupResourcesSettings{
		MemoryRequestMiB:           memoryRequestMiB,
		MemoryLimitMiB:             memoryLimitMiB,
		MemoryLimiterLimitMiB:      memoryLimiterLimitMiB,
		MemoryLimiterSpikeLimitMiB: memoryLimiterSpikeLimitMiB,
		GomemlimitMiB:              gomemlimitMiB,
		CpuRequestMillicores:       cpuRequestm,
		CpuLimitMillicores:         cpuLimitm,
	}
}

func calculateSpanMetricsEnabled(userSettings *bool, destinationTypeManifest destinations.Destination) bool {
	if userSettings == nil {
		return destinationTypeManifest.Spec.Signals.Metrics.SpanMetricsEnabledByDefault
	}
	return *userSettings
}

func getHostMetricsConfiguration(odigosConfiguration *common.OdigosConfiguration) *common.MetricsSourceHostMetricsConfiguration {

	var hostMetricsCopy common.MetricsSourceHostMetricsConfiguration
	if odigosConfiguration.MetricsSources != nil && odigosConfiguration.MetricsSources.HostMetrics != nil {
		hostMetricsCopy = *odigosConfiguration.MetricsSources.HostMetrics
	}

	if hostMetricsCopy.Disabled != nil && *hostMetricsCopy.Disabled {
		return nil
	}

	// defaults
	if hostMetricsCopy.Interval == "" {
		hostMetricsCopy.Interval = "10s"
	}

	return &hostMetricsCopy
}

func getKubeletStatsConfiguration(odigosConfiguration *common.OdigosConfiguration) *common.MetricsSourceKubeletStatsConfiguration {

	var kubeletStatsCopy common.MetricsSourceKubeletStatsConfiguration
	if odigosConfiguration.MetricsSources != nil && odigosConfiguration.MetricsSources.KubeletStats != nil {
		kubeletStatsCopy = *odigosConfiguration.MetricsSources.KubeletStats
	}

	if kubeletStatsCopy.Disabled != nil && *kubeletStatsCopy.Disabled {
		return nil
	}

	// defaults
	if kubeletStatsCopy.Interval == "" {
		kubeletStatsCopy.Interval = "10s"
	}

	return &kubeletStatsCopy
}

func getSpanMetricsConfiguration(odigosConfiguration *common.OdigosConfiguration) *common.MetricsSourceSpanMetricsConfiguration {

	var spanMetricsCopy common.MetricsSourceSpanMetricsConfiguration
	if odigosConfiguration.MetricsSources != nil && odigosConfiguration.MetricsSources.SpanMetrics != nil {
		spanMetricsCopy = *odigosConfiguration.MetricsSources.SpanMetrics
	}

	if spanMetricsCopy.Disabled != nil && *spanMetricsCopy.Disabled {
		return nil
	}

	// defaults
	if spanMetricsCopy.Interval == "" {
		spanMetricsCopy.Interval = "60s"
	}
	if spanMetricsCopy.MetricsExpiration == "" {
		spanMetricsCopy.MetricsExpiration = "5m"
	}
	if len(spanMetricsCopy.ExplicitHistogramBuckets) == 0 {
		spanMetricsCopy.ExplicitHistogramBuckets = []string{"2ms", "4ms", "6ms", "8ms", "10ms", "50ms", "100ms", "200ms", "400ms", "800ms", "1s", "1400ms", "2s", "5s", "10s", "15s"}
	}

	return &spanMetricsCopy
}

func updateMetricsSettingsForDestination(metricsConfig *odigosv1.CollectorsGroupMetricsCollectionSettings, odigosConfiguration *common.OdigosConfiguration, destination odigosv1.Destination, destinationTypeManifest destinations.Destination) {

	metricsSettings := destination.Spec.MetricsSettings
	if metricsSettings == nil {
		// apply those that are enabled by default if no settings are set
		// consider making it a global configuration in the future
		metricsConfig.AgentsTelemetry = &odigosv1.AgentsTelemetrySettings{}
		metricsConfig.HostMetrics = getHostMetricsConfiguration(odigosConfiguration)
		metricsConfig.KubeletStats = getKubeletStatsConfiguration(odigosConfiguration)
		if calculateSpanMetricsEnabled(nil, destinationTypeManifest) {
			metricsConfig.SpanMetrics = getSpanMetricsConfiguration(odigosConfiguration)
		}
		return
	}

	// is span metrics not set, use the destination manifest default
	if calculateSpanMetricsEnabled(metricsSettings.CollectSpanMetrics, destinationTypeManifest) {
		metricsConfig.SpanMetrics = getSpanMetricsConfiguration(odigosConfiguration)
	}

	// default host metrics collection to "true"
	if metricsSettings.CollectHostMetrics == nil || *metricsSettings.CollectHostMetrics {
		metricsConfig.HostMetrics = getHostMetricsConfiguration(odigosConfiguration)
	}
	// default kubelet stats collection to "true"
	if metricsSettings.CollectKubeletStats == nil || *metricsSettings.CollectKubeletStats {
		metricsConfig.KubeletStats = getKubeletStatsConfiguration(odigosConfiguration)
	}
	// default odigos own metrics collection to "false" unless explicitly enabled
	if metricsSettings.CollectOdigosOwnMetrics != nil && *metricsSettings.CollectOdigosOwnMetrics {
		metricsConfig.OdigosOwnMetrics = &odigosv1.OdigosOwnMetricsSettings{}
	}
	// default agents telemetry collection to "false"
	if metricsSettings.CollectAgentsTelemetry == nil || *metricsSettings.CollectAgentsTelemetry {
		metricsConfig.AgentsTelemetry = &odigosv1.AgentsTelemetrySettings{}
	}
}

func newNodeCollectorGroup(odigosConfiguration common.OdigosConfiguration, allDestinations odigosv1.DestinationList) *odigosv1.CollectorsGroup {

	var metricsConfig *odigosv1.CollectorsGroupMetricsCollectionSettings

	for _, destination := range allDestinations.Items {
		if destination.Spec.Disabled != nil && *destination.Spec.Disabled {
			// skip disabled destinations
			continue
		}

		destinationType := string(destination.Spec.Type)
		destinationTypeManifest, ok := destinations.GetDestinationByType(destinationType)
		if !ok {
			// ignore unknown destinations here for now (should not happen)
			continue
		}

		if !slices.Contains(destination.Spec.Signals, common.MetricsObservabilitySignal) {
			continue
		}

		if metricsConfig == nil {
			// setting it to non null is an indicator that metrics are enabled
			metricsConfig = &odigosv1.CollectorsGroupMetricsCollectionSettings{}
		}
		updateMetricsSettingsForDestination(metricsConfig, &odigosConfiguration, destination, destinationTypeManifest)
	}

	ownMetricsPort := k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault
	if odigosConfiguration.CollectorNode != nil && odigosConfiguration.CollectorNode.CollectorOwnMetricsPort != 0 {
		ownMetricsPort = odigosConfiguration.CollectorNode.CollectorOwnMetricsPort
	}

	k8sNodeLogsDirectory := ""
	if odigosConfiguration.CollectorNode != nil && odigosConfiguration.CollectorNode.K8sNodeLogsDirectory != "" {
		k8sNodeLogsDirectory = odigosConfiguration.CollectorNode.K8sNodeLogsDirectory
	}

	otlpExporterConfiguration := odigosConfiguration.CollectorNode.OtlpExporterConfiguration
	// TODO: remove after sometime it is a temporary workaround to support the deprecated field CollectorNode.EnableDataCompression
	// which replaced with OtlpExporterConfiguration.EnableDataCompression.
	if otlpExporterConfiguration == nil {
		otlpExporterConfiguration = &common.OtlpExporterConfiguration{
			EnableDataCompression: odigosConfiguration.CollectorNode.EnableDataCompression,
		}
	}

	return &odigosv1.CollectorsGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CollectorsGroup",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorCollectorGroupName,
			Namespace: env.GetCurrentNamespace(),
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role:                      odigosv1.CollectorsGroupRoleNodeCollector,
			CollectorOwnMetricsPort:   ownMetricsPort,
			K8sNodeLogsDirectory:      k8sNodeLogsDirectory,
			ResourcesSettings:         getResourceSettings(odigosConfiguration),
			OtlpExporterConfiguration: otlpExporterConfiguration,
			Metrics:                   metricsConfig,
		},
	}
}

func sync(ctx context.Context, c client.Client, scheme *runtime.Scheme) error {

	namespace := env.GetCurrentNamespace()

	var instrumentedConfigs odigosv1.InstrumentationConfigList
	err := c.List(ctx, &instrumentedConfigs)
	if err != nil {
		return errors.Join(errors.New("failed to list InstrumentationConfigs"), err)
	}
	numberOfInstrumentedApps := len(instrumentedConfigs.Items)

	if numberOfInstrumentedApps == 0 {
		// TODO: should we delete the collector group if cluster collector is not ready?
		return k8sutils.DeleteCollectorGroup(ctx, c, namespace, k8sconsts.OdigosNodeCollectorCollectorGroupName)
	}

	clusterCollectorGroup, err := k8sutils.GetCollectorGroup(ctx, c, namespace, k8sconsts.OdigosClusterCollectorCollectorGroupName)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	odigosConfiguration, err := k8sutils.GetCurrentOdigosConfiguration(ctx, c)
	if err != nil {
		return err
	}

	allDestinations := odigosv1.DestinationList{}
	err = c.List(ctx, &allDestinations)
	if err != nil {
		return err // list will return empty list if no destinations are found and not error
	}

	nodeCollectorGroup := newNodeCollectorGroup(odigosConfiguration, allDestinations)
	err = utils.SetOwnerControllerToSchedulerDeployment(ctx, c, nodeCollectorGroup, scheme)
	if err != nil {
		return err
	}

	clusterCollectorReady := clusterCollectorGroup.Status.Ready
	if clusterCollectorReady {
		return k8sutils.ApplyCollectorGroup(ctx, c, nodeCollectorGroup)
	}

	return nil
}
