package server

import (
	"context"
	"sync"
	"time"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/opampserver/pkg/agent"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	httptransport "github.com/odigos-io/odigos/opampserver/pkg/transport/http"
	unixtransport "github.com/odigos-io/odigos/opampserver/pkg/transport/unix"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func StartOpAmpServer(ctx context.Context, mgr ctrl.Manager, kubeClientSet *kubernetes.Clientset, nodeName string, odigosNs string) error {
	logger := commonlogger.LoggerCompat().With("subsystem", "opampserver")
	logger.Info("Starting opamp server")

	connectionCache := connection.NewConnectionsCache()
	sdkConfig := sdkconfig.NewSdkConfigManager(mgr, connectionCache, odigosNs)

	handlers := &ConnectionHandlers{
		logger:        commonlogger.LoggerCompat().With("subsystem", "opamphandlers"),
		sdkConfig:     sdkConfig,
		kubeclient:    mgr.GetClient(),
		kubeClientSet: kubeClientSet,
		scheme:        mgr.GetScheme(),
		nodeName:      nodeName,
	}

	updateChannel := make(chan InstrumentationUpdateTask, 1000)
	processor := NewMessageProcessor(handlers, connectionCache, updateChannel)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		ProcessInstrumentationUpdates(ctx, updateChannel, handlers)
	}()

	// Run HTTP and Unix listeners in parallel; both use the same MessageProcessor and connection cache.
	// Agents choose transport via injected env: ODIGOS_OPAMP_SERVER_HOST (HTTP) or ODIGOS_OPAMP_UNIX_SOCKET (Unix).
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httptransport.NewServer().Start(ctx, processor); err != nil && ctx.Err() == nil {
			logger.Error("OpAMP HTTP transport exited with error", "err", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := unixtransport.NewServer().Start(ctx, processor); err != nil && ctx.Err() == nil {
			logger.Error("OpAMP Unix transport exited with error", "err", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(connection.HeartbeatInterval)
		defer ticker.Stop() // Clean up when done

		for {
			select {
			case <-ctx.Done():
				// Close the updateChannel here so the worker goroutine exits.
				// HTTP graceful shutdown (and its error logging) runs in the HTTP transport goroutine.
				close(updateChannel)
				logger.Info("Shutting down live connections timeout monitor")
				return
			case <-ticker.C:
				// Clean up stale connections
				deadConnections := connectionCache.CleanupStaleConnections()
				for _, conn := range deadConnections {
					select {
					case updateChannel <- InstrumentationUpdateTask{ctx, DeleteInstance, &protobufs.AgentToServer{}, &conn}:
					default:
						logger.Error("Update channel is full, dropping task")
					}
				}
			}
		}
	}()

	wg.Wait()
	return nil
}

type InstrumentationUpdateTask struct {
	ctx            context.Context
	taskType       InstrumentationTaskType
	agentToServer  *protobufs.AgentToServer
	connectionInfo *connection.ConnectionInfo
}

type InstrumentationTaskType int

const (
	UpdateInstance InstrumentationTaskType = iota
	DeleteInstance
)

func ProcessInstrumentationUpdates(ctx context.Context, updateChannel chan InstrumentationUpdateTask, handlers *ConnectionHandlers) {
	logger := commonlogger.LoggerCompat().With("subsystem", "opampserver")
	logger.Info("Starting instrumentation instance update worker")

	for task := range updateChannel {
		switch task.taskType {
		case UpdateInstance:
			err := handlers.UpdateInstrumentationInstanceStatus(task.ctx, task.agentToServer, task.connectionInfo)
			if err != nil {
				logger.Error("Failed to update instrumentation instance", "err", err)
			}
		case DeleteInstance:
			if task.connectionInfo.Status == agent.HealthStatusNoConnectionToOpAMPServer {
				logger.Info("Skipping deletion of instrumentation instance on connection failure to opamp server", "connectionInfo", task.connectionInfo)
				continue
			}
			err := instrumentation_instance.DeleteInstrumentationInstance(ctx, task.connectionInfo.Pod, task.connectionInfo.ContainerName,
				handlers.kubeclient, int(task.connectionInfo.Pid))
			if err != nil {
				logger.Error("failed to delete instrumentation instance on connection timedout", "err", err)
			}
		default:
			logger.Error("Unknown task type received", "taskType", task.taskType)
		}
	}
	logger.Info("Shutting down instrumentation update worker")
}
