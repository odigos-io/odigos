package opampserver

import (
	"context"

	"github.com/open-telemetry/opamp-go/server"
	ctrl "sigs.k8s.io/controller-runtime"
)

// TODO: use zap logger
type Logger struct{}

func (l *Logger) Debugf(ctx context.Context, format string, v ...interface{}) {
	println("DEBUG: ", format, v)
}

func (l *Logger) Errorf(ctx context.Context, format string, v ...interface{}) {
	println("ERROR: ", format, v)
}

func StartOpAmpServer(ctx context.Context, mgr ctrl.Manager) error {
	opampsrv := server.New(&Logger{})
	err := opampsrv.Start(server.StartSettings{
		Settings: server.Settings{
			Callbacks: &K8sCrdCallbacks{
				kubeclient: mgr.GetClient(),
			},
		},
		ListenEndpoint: "0.0.0.0:4320",
		TLSConfig:      nil,
	})
	if err != nil {
		return err
	}

	return nil
}
