package main

import (
	"os"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/deviceplugin/pkg"
	"k8s.io/klog/v2"

	_ "net/http/pprof"
)

func main() {
	commonlogger.Init(os.Getenv("ODIGOS_LOG_LEVEL"), "deviceplugin")
	klog.SetLogger(commonlogger.ToLogr())
	logger := commonlogger.LoggerCompat().With("subsystem", "startup")

	dp := pkg.New()

	if err := dp.Run(); err != nil {
		logger.Error("Device plugin exited with error", "err", err)
	}
}
