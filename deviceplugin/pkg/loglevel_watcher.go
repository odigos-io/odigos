package pkg

import (
	"context"
	"time"

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
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{})
			if err != nil {
				continue
			}
			if cm.Data == nil || cm.Data[consts.OdigosConfigurationFileName] == "" {
				continue
			}
			var odigosConfig common.OdigosConfiguration
			if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &odigosConfig); err != nil {
				continue
			}
			level := "info"
			if odigosConfig.ComponentLogLevels != nil {
				level = odigosConfig.ComponentLogLevels.Resolve("odiglet")
			}
			if level != lastLevel {
				commonlogger.SetLevel(level)
				lastLevel = level
			}
		}
	}
}
