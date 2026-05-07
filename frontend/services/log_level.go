package services

import (
	"context"

	"github.com/odigos-io/odigos/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetComponentLogLevel(ctx context.Context, c client.Client, component string, level common.OdigosLogLevel) error {
	err := upsertLocalUiConfig(ctx, c, func(cfg *common.OdigosConfiguration) {
		if cfg.ComponentLogLevels == nil {
			cfg.ComponentLogLevels = &common.ComponentLogLevels{}
		}
		setComponentLogLevelField(cfg.ComponentLogLevels, component, level)
	})
	if err != nil {
		return err
	}
	if component == "ui" || component == "" {
		commonlogger.SetLevel(string(level))
	}
	return nil
}

func setComponentLogLevelField(c *common.ComponentLogLevels, component string, level common.OdigosLogLevel) {
	if c == nil {
		return
	}
	switch component {
	case "autoscaler":
		c.Autoscaler = level
	case "scheduler":
		c.Scheduler = level
	case "instrumentor":
		c.Instrumentor = level
	case "odiglet":
		c.Odiglet = level
	case "deviceplugin":
		c.Deviceplugin = level
	case "ui":
		c.UI = level
	case "collector":
		c.Collector = level
	default:
		c.Default = level
	}
}
