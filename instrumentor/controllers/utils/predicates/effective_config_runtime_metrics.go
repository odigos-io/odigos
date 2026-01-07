package predicates

import (
	"reflect"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/yaml"
)

// EffectiveConfigRuntimeMetricsChangedPredicate is a predicate that checks if the runtime metrics
// configuration in the effective config has changed.
//
// It only triggers when the runtimeMetrics section of the effective configuration changes,
// avoiding unnecessary reconciliations for other config changes.
type EffectiveConfigRuntimeMetricsChangedPredicate struct{}

var _ predicate.Predicate = &EffectiveConfigRuntimeMetricsChangedPredicate{}

// extractRuntimeMetricsConfig extracts the runtime metrics configuration from a ConfigMap
func (p EffectiveConfigRuntimeMetricsChangedPredicate) extractRuntimeMetricsConfig(configMap *corev1.ConfigMap) *common.MetricsSourceAgentRuntimeMetricsConfiguration {
	if configMap == nil || configMap.Data == nil {
		return nil
	}

	configData, exists := configMap.Data[consts.OdigosConfigurationFileName]
	if !exists {
		return nil
	}

	var odigosConfig common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(configData), &odigosConfig); err != nil {
		return nil
	}

	if odigosConfig.MetricsSources == nil ||
		odigosConfig.MetricsSources.AgentMetrics == nil ||
		odigosConfig.MetricsSources.AgentMetrics.RuntimeMetrics == nil {
		return nil
	}

	return odigosConfig.MetricsSources.AgentMetrics.RuntimeMetrics
}

func (p EffectiveConfigRuntimeMetricsChangedPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	configMap, ok := e.Object.(*corev1.ConfigMap)
	if !ok {
		return false
	}

	// Only filter for odigos-effective-config
	if configMap.Name != consts.OdigosEffectiveConfigName {
		return false
	}

	// Trigger if runtime metrics config exists on creation
	return p.extractRuntimeMetricsConfig(configMap) != nil
}

func (p EffectiveConfigRuntimeMetricsChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldConfigMap, oldOk := e.ObjectOld.(*corev1.ConfigMap)
	newConfigMap, newOk := e.ObjectNew.(*corev1.ConfigMap)

	if !oldOk || !newOk {
		return false
	}

	// Only filter for odigos-effective-config
	if oldConfigMap.Name != consts.OdigosEffectiveConfigName || newConfigMap.Name != consts.OdigosEffectiveConfigName {
		return false
	}

	oldRuntimeMetrics := p.extractRuntimeMetricsConfig(oldConfigMap)
	newRuntimeMetrics := p.extractRuntimeMetricsConfig(newConfigMap)

	// Trigger if runtime metrics configuration has changed
	return !reflect.DeepEqual(oldRuntimeMetrics, newRuntimeMetrics)
}

func (p EffectiveConfigRuntimeMetricsChangedPredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		return false
	}

	configMap, ok := e.Object.(*corev1.ConfigMap)
	if !ok {
		return false
	}

	// Only filter for odigos-effective-config
	if configMap.Name != consts.OdigosEffectiveConfigName {
		return false
	}

	// Trigger if runtime metrics config existed before deletion
	return p.extractRuntimeMetricsConfig(configMap) != nil
}

func (p EffectiveConfigRuntimeMetricsChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}
