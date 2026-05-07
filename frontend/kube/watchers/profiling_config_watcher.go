package watchers

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/frontend/services/profiles"

	corev1 "k8s.io/api/core/v1"
	toolscache "k8s.io/client-go/tools/cache"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
)

// StartProfilingConfigWatcher watches the effective-config ConfigMap and toggles OTLP profile ingest without restarting the UI pod.
func StartProfilingConfigWatcher(
	ctx context.Context,
	k8sCache ctrlcache.Cache,
	odigosNamespace string,
	gate *profiles.IngestGate,
	profileStore *profiles.ProfileStore,
) error {
	informer, err := k8sCache.GetInformer(ctx, &corev1.ConfigMap{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap informer for profiling: %w", err)
	}

	log := commonlogger.LoggerCompat().With("subsystem", "profiling-ingest-watch")

	reconcileProfilingIngestGate := func(cm *corev1.ConfigMap) {
		if cm.Namespace != odigosNamespace || cm.Name != consts.OdigosEffectiveConfigName {
			return
		}
		cfg, err := services.OdigosConfigurationFromConfigMap(cm)
		if err != nil {
			log.Error("effective-config parse failed; leaving profiling ingest state unchanged", "err", err)
			return
		}
		profilingEnabledNew := services.ProfilingEnabledFromOdigosConfig(cfg)
		profilingEnabledOld := gate.IsEnabled()
		gate.Set(profilingEnabledNew)
		if profilingEnabledOld && !profilingEnabledNew {
			profileStore.ClearAllSlots()
		}
		if profilingEnabledOld != profilingEnabledNew {
			log.Info("profiling OTLP ingest", "enabled", profilingEnabledNew)
		}
	}

	_, err = informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if cm, ok := configMapFromInformerObj(obj); ok {
				reconcileProfilingIngestGate(cm)
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			if cm, ok := configMapFromInformerObj(newObj); ok {
				reconcileProfilingIngestGate(cm)
			}
		},
		DeleteFunc: func(obj interface{}) {
			cm, ok := configMapFromInformerObj(obj)
			if !ok || cm.Namespace != odigosNamespace || cm.Name != consts.OdigosEffectiveConfigName {
				return
			}
			if gate.IsEnabled() {
				gate.Set(false)
				profileStore.ClearAllSlots()
				log.Info("profiling OTLP ingest disabled (effective-config deleted)")
			}
		},
	})
	if err != nil {
		return fmt.Errorf("failed to register profiling ingest ConfigMap handler: %w", err)
	}
	return nil
}

// configMapFromInformerObj unwraps a ConfigMap from an informer callback object or a delete tombstone.
func configMapFromInformerObj(obj interface{}) (*corev1.ConfigMap, bool) {
	cm, ok := obj.(*corev1.ConfigMap)
	if ok {
		return cm, true
	}
	tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
	if !ok {
		return nil, false
	}
	cm, ok = tombstone.Obj.(*corev1.ConfigMap)
	if !ok {
		return nil, false
	}
	return cm, true
}
