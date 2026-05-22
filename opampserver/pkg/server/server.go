package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	commonopamp "github.com/odigos-io/odigos/common/opamp"
	"github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/opampserver/pkg/agent"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig"
	"github.com/odigos-io/odigos/opampserver/pkg/transport"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

func StartOpAmpServer(ctx context.Context, mgr ctrl.Manager, kubeClientSet *kubernetes.Clientset, nodeName string, odigosNs string) error {
	logger := commonlogger.LoggerCompat().With("subsystem", "opampserver")

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

	// Buffered channel for instrumentation instances updates
	updateChannel := make(chan InstrumentationUpdateTask, 1000)
	processor := NewMessageProcessor(handlers, connectionCache, updateChannel)

	var wg sync.WaitGroup

	// Start the worker goroutine to process instrumentation instances updates sequentially
	wg.Add(1)
	go func() {
		defer wg.Done()
		ProcessInstrumentationUpdates(ctx, updateChannel, handlers)
	}()

	// Both listeners speak plain HTTP/1.1 over their respective transports and share
	// the same MessageProcessor + connection cache. Agents choose via injected env:
	//   ODIGOS_OPAMP_SERVER_HOST    (HTTP over TCP)
	//   ODIGOS_OPAMP_UNIX_SOCKET    (HTTP over node-local unix socket)
	tcpServer := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", commonconsts.OpAMPPort),
		Handler: transport.NewHandler(ctx, processor, commonopamp.OpAmpTransportHTTP),
	}
	unixServer := &http.Server{
		Handler: transport.NewHandler(ctx, processor, commonopamp.OpAmpTransportUnix),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting opamp HTTP server", "listenEndpoint", tcpServer.Addr)
		if err := tcpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("OpAMP HTTP transport exited with error", "err", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting opamp Unix server", "socket", k8sconsts.OdigosOpampUnixSocketPath)
		if err := serveUnix(ctx, unixServer, logger); err != nil && ctx.Err() == nil {
			logger.Error("OpAMP Unix transport exited with error", "err", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tcpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shut down HTTP opamp server", "err", err)
		}
		if err := unixServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shut down Unix opamp server", "err", err)
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
				// Close the updateChannel here so the worker goroutine exits
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

// serveUnix starts a unix-socket HTTP listener under /var/odigos/exchange/exchange.sock.
// The socket file is recreated on startup and removed on shutdown.
func serveUnix(ctx context.Context, srv *http.Server, logger *commonlogger.OdigosLogger) error {
	socketPath := k8sconsts.OdigosOpampUnixSocketPath
	if err := os.MkdirAll(k8sconsts.OdigosOpampExchangeDir, 0o755); err != nil {
		return fmt.Errorf("mkdir exchange dir: %w", err)
	}
	_ = os.Remove(socketPath)

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", socketPath, err)
	}
	defer func() {
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			logger.Error("Failed to remove Unix opamp socket", "err", err, "socket", socketPath)
		}
	}()
	if err := os.Chmod(socketPath, 0o666); err != nil {
		logger.Error("Failed to chmod unix socket", "err", err, "socket", socketPath)
	}

	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
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
		// Do not delete the instrumentation instance if the connection failed;
		// Instead, retain it in an unhealthy state so the UI can display relevant information.
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
