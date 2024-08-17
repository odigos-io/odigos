package server

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/deviceid"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConnectionHandlers struct {
	deviceIdCache *deviceid.DeviceIdCache
	sdkConfig     *sdkconfig.SdkConfigManager
	logger        logr.Logger
	kubeclient    client.Client
	scheme        *runtime.Scheme // TODO: revisit this, we should not depend on controller runtime
	nodeName      string
}

func (c *ConnectionHandlers) OnNewConnection(ctx context.Context, deviceId string, firstMessage *protobufs.AgentToServer) (*connection.ConnectionInfo, *protobufs.ServerToAgent, error) {

	if firstMessage.AgentDescription == nil {
		// first message must be agent description.
		// it is, however, possible that the OpAMP server restarted, and the agent is trying to reconnect.
		// in which case we send back flag and request full status update.
		c.logger.Info("Agent description is missing in the first OpAMP message, requesting full state update", "deviceId", deviceId)
		serverToAgent := &protobufs.ServerToAgent{
			Flags: uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState),
		}
		return nil, serverToAgent, nil
	}

	var pid int64
	for _, attr := range firstMessage.AgentDescription.IdentifyingAttributes {
		if attr.Key == string(semconv.ProcessPIDKey) {
			pid = attr.Value.GetIntValue()
			break
		}
	}
	if pid == 0 {
		return nil, nil, fmt.Errorf("missing pid in agent description")
	}

	k8sAttributes, pod, err := c.deviceIdCache.GetAttributesFromDevice(ctx, deviceId)
	if err != nil {
		c.logger.Error(err, "failed to get attributes from device", "deviceId", deviceId)
		return nil, nil, err
	}

	podWorkload := workload.PodWorkload{
		Namespace: pod.GetNamespace(),
		Kind:      k8sAttributes.WorkloadKind,
		Name:      k8sAttributes.WorkloadName,
	}

	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(k8sAttributes.WorkloadName, k8sAttributes.WorkloadKind)
	remoteResourceAttributes, err := configresolvers.CalculateServerAttributes(k8sAttributes, c.nodeName)
	if err != nil {
		c.logger.Error(err, "failed to calculate server attributes", "k8sAttributes", k8sAttributes)
		return nil, nil, err
	}

	fullRemoteConfig, err := c.sdkConfig.GetFullConfig(ctx, remoteResourceAttributes, &podWorkload, instrumentedAppName)
	if err != nil {
		c.logger.Error(err, "failed to get full config", "k8sAttributes", k8sAttributes)
		return nil, nil, err
	}
	c.logger.Info("new OpAMP client connected", "deviceId", deviceId, "namespace", k8sAttributes.Namespace, "podName", k8sAttributes.PodName, "instrumentedAppName", instrumentedAppName, "workloadKind", k8sAttributes.WorkloadKind, "workloadName", k8sAttributes.WorkloadName, "containerName", k8sAttributes.ContainerName, "otelServiceName", k8sAttributes.OtelServiceName)

	connectionInfo := &connection.ConnectionInfo{
		DeviceId:                 deviceId,
		Workload:                 podWorkload,
		Pod:                      pod,
		ContainerName:            k8sAttributes.ContainerName,
		Pid:                      pid,
		InstrumentedAppName:      instrumentedAppName,
		AgentRemoteConfig:        fullRemoteConfig,
		RemoteResourceAttributes: remoteResourceAttributes,
	}

	serverToAgent := &protobufs.ServerToAgent{
		RemoteConfig: fullRemoteConfig,
	}

	return connectionInfo, serverToAgent, nil
}

func (c *ConnectionHandlers) OnAgentToServerMessage(ctx context.Context, request *protobufs.AgentToServer, connectionInfo *connection.ConnectionInfo) (*protobufs.ServerToAgent, error) {
	response := protobufs.ServerToAgent{}

	// If the remote config changed, send the new config to the agent on the response
	if request.RemoteConfigStatus == nil {
		// this is to support older agents which do not send remote config status
		c.logger.Info("missing remote config status in agent to server message", "workload", connectionInfo.Workload)
	} else {
		if !bytes.Equal(request.RemoteConfigStatus.LastRemoteConfigHash, connectionInfo.AgentRemoteConfig.ConfigHash) {
			c.logger.Info("Remote config changed, sending new config to agent", "workload", connectionInfo.Workload)
			response.RemoteConfig = connectionInfo.AgentRemoteConfig
		}
	}

	return &response, nil
}

func (c *ConnectionHandlers) OnConnectionClosed(ctx context.Context, connectionInfo *connection.ConnectionInfo) {
	// keep the instrumentation instance CR in unhealthy state so it can be used for troubleshooting
}

func (c *ConnectionHandlers) OnConnectionNoHeartbeat(ctx context.Context, connectionInfo *connection.ConnectionInfo) error {
	healthy := false
	message := fmt.Sprintf("OpAMP server did not receive heartbeat from the agent, last message time: %s", connectionInfo.LastMessageTime.Format("2006-01-02 15:04:05 MST"))
	// keep the instrumentation instance CR in unhealthy state so it can be used for troubleshooting
	err := instrumentation_instance.UpdateInstrumentationInstanceStatus(ctx, connectionInfo.Pod, connectionInfo.ContainerName, c.kubeclient, connectionInfo.InstrumentedAppName, int(connectionInfo.Pid), c.scheme,
		instrumentation_instance.WithHealthy(&healthy, string(common.AgentHealthStatusNoHeartbeat), &message),
	)
	if err != nil {
		return fmt.Errorf("failed to persist instrumentation instance health status on connection timedout: %w", err)
	}

	return nil
}

func (c *ConnectionHandlers) UpdateInstrumentationInstanceStatus(ctx context.Context, message *protobufs.AgentToServer, connectionInfo *connection.ConnectionInfo) error {

	isAgentDisconnect := message.AgentDisconnect != nil
	hasHealth := message.Health != nil
	// when agent disconnects, it need to report that it is unhealthy and disconnected
	if isAgentDisconnect {
		if !hasHealth {
			return fmt.Errorf("missing health in agent disconnect message")
		}
		if message.Health.Healthy {
			return fmt.Errorf("agent disconnect message with healthy status")
		}
		if message.Health.LastError == "" {
			return fmt.Errorf("missing last error in unhealthy message")
		}
	}

	dynamicOptions := make([]instrumentation_instance.InstrumentationInstanceOption, 0)

	if message.AgentDescription != nil {
		identifyingAttributes := make([]odigosv1.Attribute, 0, len(message.AgentDescription.IdentifyingAttributes))
		for _, attr := range message.AgentDescription.IdentifyingAttributes {
			strValue := ConvertAnyValueToString(attr.GetValue())
			identifyingAttributes = append(identifyingAttributes, odigosv1.Attribute{
				Key:   attr.Key,
				Value: strValue,
			})
		}
		dynamicOptions = append(dynamicOptions, instrumentation_instance.WithAttributes(identifyingAttributes, []odigosv1.Attribute{}))
	}

	// agent is only expected to send health status when it changes, so if found - persist it to CRD as new status
	if hasHealth {
		// always record healthy status into the CRD, to reflect the current state
		healthy := message.Health.Healthy
		dynamicOptions = append(dynamicOptions, instrumentation_instance.WithHealthy(&healthy, message.Health.Status, &message.Health.LastError))
	}

	if len(dynamicOptions) > 0 {
		err := instrumentation_instance.UpdateInstrumentationInstanceStatus(ctx, connectionInfo.Pod, connectionInfo.ContainerName, c.kubeclient, connectionInfo.InstrumentedAppName, int(connectionInfo.Pid), c.scheme, dynamicOptions...)
		if err != nil {
			return fmt.Errorf("failed to persist instrumentation instance status: %w", err)
		}
	}

	return nil
}
