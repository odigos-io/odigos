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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	podWorkload := common.PodWorkload{
		Namespace: pod.GetNamespace(),
		Kind:      k8sAttributes.WorkloadKind,
		Name:      k8sAttributes.WorkloadName,
	}

	instrumentedAppName := workload.GetRuntimeObjectName(k8sAttributes.WorkloadName, k8sAttributes.WorkloadKind)
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
	c.logger.Info("Connection closed for device", "deviceId", connectionInfo.DeviceId)
	instrumentationInstanceName := instrumentation_instance.InstrumentationInstanceName(connectionInfo.Pod, int(connectionInfo.Pid))
	err := c.kubeclient.Delete(ctx, &odigosv1.InstrumentationInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "InstrumentationInstance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instrumentationInstanceName,
			Namespace: connectionInfo.Pod.GetNamespace(),
		},
	})

	if err != nil && !apierrors.IsNotFound(err) {
		c.logger.Error(err, "failed to delete instrumentation instance", "instrumentationInstanceName", instrumentationInstanceName)
	}
}

func (c *ConnectionHandlers) PersistInstrumentationDeviceStatus(ctx context.Context, message *protobufs.AgentToServer, connectionInfo *connection.ConnectionInfo) error {
	if message.AgentDescription != nil {
		identifyingAttributes := make([]odigosv1.Attribute, 0, len(message.AgentDescription.IdentifyingAttributes))
		for _, attr := range message.AgentDescription.IdentifyingAttributes {
			strValue := ConvertAnyValueToString(attr.GetValue())
			identifyingAttributes = append(identifyingAttributes, odigosv1.Attribute{
				Key:   attr.Key,
				Value: strValue,
			})
		}

		healthy := true // TODO: populate this field with real health status
		err := instrumentation_instance.PersistInstrumentationInstanceStatus(ctx, connectionInfo.Pod, c.kubeclient, connectionInfo.InstrumentedAppName, int(connectionInfo.Pid), c.scheme,
			instrumentation_instance.WithIdentifyingAttributes(identifyingAttributes),
			instrumentation_instance.WithMessage("Agent connected"),
			instrumentation_instance.WithHealthy(&healthy),
		)
		if err != nil {
			return fmt.Errorf("failed to persist instrumentation instance status: %w", err)
		}
	}

	return nil
}
