package server

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/opampserver/pkg/deviceid"
	"github.com/open-telemetry/opamp-go/server"
	"k8s.io/client-go/kubernetes"
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

func StartOpAmpServer(ctx context.Context, logger logr.Logger, mgr ctrl.Manager, kubeClient *kubernetes.Clientset) error {

	listenEndpoint := "0.0.0.0:4320"
	logger.Info("Starting opamp server", "listenEndpoint", listenEndpoint)

	deviceidCache, err := deviceid.NewDeviceIdCache(logger, kubeClient)
	if err != nil {
		return err
	}

	opampsrv := server.New(&Logger{})
	err = opampsrv.Start(server.StartSettings{
		Settings: server.Settings{
			Callbacks: &K8sCrdCallbacks{
				logger:        logger,
				deviceIdCache: deviceidCache,
				kubeclient:    mgr.GetClient(),
				scheme:        mgr.GetScheme(),
			},
		},
		ListenEndpoint: listenEndpoint,
		TLSConfig:      nil,
	})
	if err != nil {
		return err
	}

	return nil
}
