package log

import (
	"flag"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

var Logger logr.Logger

func Init() error {
	zapLog, err := zap.NewProduction()
	if err != nil {
		return err
	}

	Logger = zapr.NewLogger(zapLog)

	// used by device manager logger
	flag.Parse()

	return nil
}
