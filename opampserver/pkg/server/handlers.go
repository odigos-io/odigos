package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/deviceid"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

type ConnectionHandlers struct {
	deviceIdCache *deviceid.DeviceIdCache
	logger        logr.Logger
}

func (c *ConnectionHandlers) OnNewConnection(ctx context.Context, deviceId string, firstMessage *protobufs.AgentToServer) (*ConnectionInfo, *protobufs.ServerToAgent, error) {

	k8sAttributes, pod, err := c.deviceIdCache.GetAttributesFromDevice(ctx, deviceId)
	if err != nil {
		c.logger.Error(err, "failed to get attributes from device", "deviceId", deviceId)
		return nil, nil, err
	}
	serverOfferedResourceAttributes, err := calculateServerAttributes(k8sAttributes)
	if err != nil {
		c.logger.Error(err, "failed to calculate server attributes", "deviceId", deviceId)
		return nil, nil, err
	}

	instrumentedAppName := workload.GetRuntimeObjectName(k8sAttributes.WorkloadName, k8sAttributes.WorkloadKind)
	c.logger.Info("new OpAMP client connected", "deviceId", deviceId, "namespace", k8sAttributes.Namespace, "podName", k8sAttributes.PodName, "instrumentedAppName", instrumentedAppName, "workloadKind", k8sAttributes.WorkloadKind, "workloadName", k8sAttributes.WorkloadName, "containerName", k8sAttributes.ContainerName, "otelServiceName", k8sAttributes.OtelServiceName)

	resourceAttributeConfig, err := json.Marshal(serverOfferedResourceAttributes)
	if err != nil {
		c.logger.Error(err, "failed to marshal server offered resource attributes")
		return nil, nil, err
	}

	serverAttrsRemoteCfg := protobufs.AgentRemoteConfig{
		Config: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				"server-resolved-resource-attributes": {
					Body:        resourceAttributeConfig,
					ContentType: "application/json",
				},
			},
		},
	}

	opampResponse := &protobufs.ServerToAgent{
		RemoteConfig: &serverAttrsRemoteCfg,
	}
	connectionInfo := &ConnectionInfo{
		DeviceId: deviceId,
		Pod:      pod,
	}

	return connectionInfo, opampResponse, nil
}

func (c *ConnectionHandlers) OnAgentToServerMessage(ctx context.Context, request *protobufs.AgentToServer, connectionInfo *ConnectionInfo) (*protobufs.ServerToAgent, error) {

	fmt.Println("Received message from agent", request.String())
	return &protobufs.ServerToAgent{}, nil
}

func (c *ConnectionHandlers) OnConnectionClosed(ctx context.Context, connectionInfo *ConnectionInfo) {
	fmt.Println("Connection closed for device", connectionInfo.DeviceId)
}
