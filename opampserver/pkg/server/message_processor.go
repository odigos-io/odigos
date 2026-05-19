package server

import (
	"context"

	commonopamp "github.com/odigos-io/odigos/common/opamp"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/opampserver/pkg/agent"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/opamptypes"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

// MessageProcessor runs shared OpAMP logic for every transport.
type MessageProcessor struct {
	logger          *commonlogger.OdigosLogger
	handlers        *ConnectionHandlers
	connectionCache *connection.ConnectionsCache
	updateChannel   chan InstrumentationUpdateTask
}

func NewMessageProcessor(
	handlers *ConnectionHandlers,
	connectionCache *connection.ConnectionsCache,
	updateChannel chan InstrumentationUpdateTask,
) *MessageProcessor {
	return &MessageProcessor{
		logger:          commonlogger.LoggerCompat().With("subsystem", "opampserver"),
		handlers:        handlers,
		connectionCache: connectionCache,
		updateChannel:   updateChannel,
	}
}

// Process handles a single AgentToServer message and returns ServerToAgent.
func (p *MessageProcessor) Process(ctx context.Context, agentToServer *protobufs.AgentToServer, transport commonopamp.OpAmpTransport) (*protobufs.ServerToAgent, opamptypes.ProcessStatus) {
	instanceUid := string(agentToServer.InstanceUid)
	if instanceUid == "" {
		p.logger.Error("InstanceUid is missing", "transport", transport)
		return nil, opamptypes.ProcessBadRequest
	}

	isAgentDisconnect := agentToServer.AgentDisconnect != nil

	var serverToAgent *protobufs.ServerToAgent
	var err error
	connectionInfo, exists := p.connectionCache.GetConnection(instanceUid)
	if !exists {
		connectionInfo, serverToAgent, err = p.handlers.OnNewConnection(ctx, agentToServer)
		if err != nil {
			p.logger.Error("Failed to process new connection", "err", err, "transport", transport)
			return nil, opamptypes.ProcessError
		}
		if connectionInfo != nil {
			p.connectionCache.AddConnection(instanceUid, connectionInfo)
			p.logger.Debug("new OpAMP client connected",
				"transport", transport,
				"workloadNamespace", connectionInfo.Workload.Namespace,
				"workloadName", connectionInfo.Workload.Name,
				"workloadKind", connectionInfo.Workload.Kind,
			)
		}
	} else {
		serverToAgent, err = p.handlers.OnAgentToServerMessage(ctx, agentToServer, connectionInfo)
		if err != nil {
			p.logger.Error("Failed to process opamp message", "err", err, "transport", transport)
			return nil, opamptypes.ProcessError
		}
	}

	if connectionInfo != nil && (agentToServer.AgentDescription != nil || agentToServer.Health != nil) {
		select {
		case p.updateChannel <- InstrumentationUpdateTask{ctx, UpdateInstance, agentToServer, connectionInfo}:
		default:
			p.logger.Error("Update channel is full, dropping task")
		}
	}

	if serverToAgent == nil {
		p.logger.Error("No response from opamp handler", "transport", transport)
		return nil, opamptypes.ProcessError
	}

	if isAgentDisconnect {
		if connectionInfo != nil {
			p.logger.Debug("Agent disconnected",
				"transport", transport,
				"workloadNamespace", connectionInfo.Workload.Namespace,
				"workloadName", connectionInfo.Workload.Name,
				"workloadKind", connectionInfo.Workload.Kind,
			)
		}
		p.connectionCache.RemoveConnection(instanceUid)
	} else {
		healthStatus := agent.HealthStatusUnknown
		if agentToServer.Health != nil {
			healthStatus = agent.GetAgentHealthStatus(agentToServer.Health.Status)
		}
		p.connectionCache.RecordMessageTime(instanceUid, healthStatus)
	}

	serverToAgent.InstanceUid = agentToServer.InstanceUid
	return serverToAgent, opamptypes.ProcessOK
}
