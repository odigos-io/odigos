package log

import (
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
	return nil
}
