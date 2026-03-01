package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	commonconsts "github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/opampserver/pkg/agent"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"google.golang.org/protobuf/proto"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func StartOpAmpServer(ctx context.Context, mgr ctrl.Manager, kubeClientSet *kubernetes.Clientset, nodeName string, odigosNs string) error {
	logger := commonlogger.Logger().With("subsystem", "opamp-server")
	listenEndpoint := fmt.Sprintf("0.0.0.0:%d", commonconsts.OpAMPPort)
	logger.Info("Starting opamp server", "listenEndpoint", listenEndpoint)

	connectionCache := connection.NewConnectionsCache()

	sdkConfig := sdkconfig.NewSdkConfigManager(mgr, connectionCache, odigosNs)

	handlers := &ConnectionHandlers{
		sdkConfig:     sdkConfig,
		kubeclient:    mgr.GetClient(),
		kubeClientSet: kubeClientSet,
		scheme:        mgr.GetScheme(),
		nodeName:      nodeName,
	}

	// Buffered channel for instrumentation instances updates
	updateChannel := make(chan InstrumentationUpdateTask, 1000)

	http.HandleFunc("POST /v1/opamp", func(w http.ResponseWriter, req *http.Request) {

		// we only support plain http connections.
		// this check will filter out WS connections if they arrive for any reasons.
		if req.Header.Get("Content-Type") != "application/x-protobuf" {
			http.Error(w, "Content-Type header is not application/x-protobuf", http.StatusBadRequest)
			return
		}

		bytes, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		// assume the data is not compressed, which is not relevant and not supported in odigos

		// Decode the message as a Protobuf message.
		var agentToServer protobufs.AgentToServer
		err = proto.Unmarshal(bytes, &agentToServer)
		if err != nil {
			logger.Error("Cannot decode opamp message from HTTP Body", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		instanceUid := string(agentToServer.InstanceUid)
		if instanceUid == "" {
			logger.Error("InstanceUid is missing")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		isAgentDisconnect := agentToServer.AgentDisconnect != nil

		var serverToAgent *protobufs.ServerToAgent
		connectionInfo, exists := connectionCache.GetConnection(instanceUid)
		if !exists {
			connectionInfo, serverToAgent, err = handlers.OnNewConnection(ctx, &agentToServer)
			if err != nil {
				logger.Error("Failed to process new connection", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if connectionInfo != nil {
				connectionCache.AddConnection(instanceUid, connectionInfo)
			}
		} else {
			serverToAgent, err = handlers.OnAgentToServerMessage(ctx, &agentToServer, connectionInfo)
			if err != nil {
				logger.Error("Failed to process opamp message", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		// Only update the InstrumentationInstance if the message contains the relevant data
		// This is to avoid unnecessary updates when the message is a heartbeat
		if connectionInfo != nil && (agentToServer.AgentDescription != nil || agentToServer.Health != nil) {
			select {
			case updateChannel <- InstrumentationUpdateTask{ctx, UpdateInstance, &agentToServer, connectionInfo}:
			default:
				logger.Error("Update channel is full, dropping task")
			}
		}

		if serverToAgent == nil {
			logger.Error("No response from opamp handler", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if isAgentDisconnect {

			// This may occurs when Odiglet restarts, and a previously connected pod sends a disconnect message right after reconnecting.
			if connectionInfo != nil {
				logger.Info("Agent disconnected", "workloadNamespace", connectionInfo.Workload.Namespace, "workloadName", connectionInfo.Workload.Name, "workloadKind", connectionInfo.Workload.Kind)
			}
			// if agent disconnects, remove the connection from the cache
			// as it is not expected to send additional messages
			connectionCache.RemoveConnection(instanceUid)
		} else {
			// keep record in memory of last message time, to detect stale connections
			// Check if agentToServer.Health is nil; if so, use HealthStatusUnknown
			healthStatus := agent.HealthStatusUnknown
			if agentToServer.Health != nil {
				healthStatus = agent.GetAgentHealthStatus(agentToServer.Health.Status)
			}
			connectionCache.RecordMessageTime(instanceUid, healthStatus)
		}

		serverToAgent.InstanceUid = agentToServer.InstanceUid

		// Marshal the response.
		bytes, err = proto.Marshal(serverToAgent)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Send the response.
		w.Header().Set("Content-Type", "application/x-protobuf")
		_, err = w.Write(bytes)

		if err != nil {
			logger.Error("Failed to write response", "err", err)
		}
	})

	server := &http.Server{Addr: listenEndpoint, Handler: nil}
	var wg sync.WaitGroup

	// Start the worker goroutine to process instrumentation instances updates sequentially
	wg.Add(1)
	go func() {
		defer wg.Done()
		ProcessInstrumentationUpdates(ctx, updateChannel, handlers)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Error starting opamp server", "err", err)
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

				if err := server.Shutdown(ctx); err != nil {
					logger.Error("Failed to shut down the http server for incoming connections", "err", err)
				}
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
	logger := commonlogger.Logger().With("subsystem", "opamp-server")
	logger.Info("Starting instrumentation instance update worker")

	for task := range updateChannel {
		switch task.taskType {
		case UpdateInstance:
			err := handlers.UpdateInstrumentationInstanceStatus(task.ctx, task.agentToServer, task.connectionInfo)
			if err != nil {
				logger.Error("Failed to update instrumentation instance", "err", err)
			}
		case DeleteInstance:
			// Do not delete the instrumentation instance if the connection failed;
			// Instead, retain it in an unhealthy state so the UI can display relevant information.
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
