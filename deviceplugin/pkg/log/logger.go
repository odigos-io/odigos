package log

import (
	"flag"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	bridge "github.com/odigos-io/opentelemetry-zap-bridge"
	"go.uber.org/zap"
)

var Logger logr.Logger

func Init() error {

	zapLogger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	zapLogger = bridge.AttachToZapLogger(zapLogger)
	Logger = zapr.NewLogger(zapLogger)

	// used by device manager logger
	flag.Parse()

	return nil
}
