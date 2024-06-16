package server

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/opampserver/pkg/deviceid"
	"github.com/open-telemetry/opamp-go/server"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

type OpAMPLogger struct {
	logger logr.Logger
}

func (l *OpAMPLogger) Debugf(ctx context.Context, format string, v ...interface{}) {
	l.logger.V(1).Info(fmt.Sprintf(format, v...))
}

func (l *OpAMPLogger) Errorf(ctx context.Context, format string, v ...interface{}) {
	l.logger.Error(fmt.Errorf(format, v...), "")
}

func StartOpAmpServer(ctx context.Context, logger logr.Logger, mgr ctrl.Manager, kubeClient *kubernetes.Clientset) error {

	listenEndpoint := "0.0.0.0:4320"
	logger.Info("Starting opamp server", "listenEndpoint", listenEndpoint)

	deviceidCache, err := deviceid.NewDeviceIdCache(logger, kubeClient)
	if err != nil {
		return err
	}

	opampsrv := server.New(&OpAMPLogger{})
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
