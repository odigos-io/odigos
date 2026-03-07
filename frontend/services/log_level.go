package services

import (
	"context"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func SetComponentLogLevel(ctx context.Context, c client.Client, component string, level common.OdigosLogLevel) error {
	ns := env.GetCurrentNamespace()
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var cm v1.ConfigMap
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosLocalUiConfigName}, &cm); err != nil {
			return err
		}
		var cfg common.OdigosConfiguration
		if cm.Data != nil && cm.Data[consts.OdigosConfigurationFileName] != "" {
			_ = yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &cfg)
		}
		if cfg.ComponentLogLevels == nil {
			cfg.ComponentLogLevels = &common.ComponentLogLevels{}
		}
		switch component {
		case "autoscaler":
			cfg.ComponentLogLevels.Autoscaler = level
		case "scheduler":
			cfg.ComponentLogLevels.Scheduler = level
		case "instrumentor":
			cfg.ComponentLogLevels.Instrumentor = level
		case "odiglet":
			cfg.ComponentLogLevels.Odiglet = level
		case "ui":
			cfg.ComponentLogLevels.UI = level
		default:
			cfg.ComponentLogLevels.Default = level
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[consts.OdigosConfigurationFileName] = string(data)
		return c.Update(ctx, &cm)
	})
	if err != nil {
		return err
	}
	if component == "ui" || component == "" {
		commonlogger.SetLevel(string(level))
	}
	return nil
}
