package pkg

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func startLogLevelWatcher(ctx context.Context) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return
	}

	ns := env.GetCurrentNamespace()
	var lastLevel string
	applyLevelFromConfigMap := func(cm *corev1.ConfigMap) {
		if cm == nil || cm.Data == nil || cm.Data[consts.OdigosConfigurationFileName] == "" {
			return
		}
		var odigosConfig common.OdigosConfiguration
		if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &odigosConfig); err != nil {
			return
		}
		level := "info"
		if odigosConfig.ComponentLogLevels != nil {
			level = odigosConfig.ComponentLogLevels.Resolve("deviceplugin")
		}
		if level != lastLevel {
			commonlogger.SetLevel(level)
			lastLevel = level
		}
	}

	// Initial read so we apply level before the first watch event
	if cm, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{}); err == nil {
		applyLevelFromConfigMap(cm)
	}

	backoff := time.Second
	for ctx.Err() == nil {
		watch, err := cs.CoreV1().ConfigMaps(ns).Watch(ctx, metav1.ListOptions{
			FieldSelector: "metadata.name=" + consts.OdigosEffectiveConfigName,
		})
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				if backoff < 30*time.Second {
					backoff *= 2
				}
			}
			continue
		}
		backoff = time.Second

		for event := range watch.ResultChan() {
			if event.Object == nil {
				continue
			}
			cm, ok := event.Object.(*corev1.ConfigMap)
			if !ok {
				continue
			}
			applyLevelFromConfigMap(cm)
		}
		watch.Stop()
	}
}
