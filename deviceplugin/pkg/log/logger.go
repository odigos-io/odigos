package log

import (
	"flag"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

var Logger logr.Logger

func Init() error {

	zapLogger, err := zap.NewProduction()
	if err != nil {
		return err
	}

	Logger = zapr.NewLogger(zapLogger)

	flag.Parse()

	return nil
}
