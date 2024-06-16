package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/instrumentation_instance"
	"github.com/odigos-io/odigos/opampserver/pkg/deviceid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	semconv "go.opentelemetry.io/collector/semconv/v1.9.0"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Callbacks is an implementation of the Callbacks interface.
var _ types.Callbacks = &K8sCrdCallbacks{}

type K8sCrdCallbacks struct {
	scheme        *runtime.Scheme
	logger        logr.Logger
	deviceIdCache *deviceid.DeviceIdCache
	kubeclient    client.Client
}

func (c *K8sCrdCallbacks) OnConnecting(request *http.Request) types.ConnectionResponse {
	deviceId, err := getDeviceIdFromHeader(request)
	if err != nil {
		return types.ConnectionResponse{
			Accept:         false,
			HTTPStatusCode: 401,
		}
	}

	k8sAttributes, pod, err := c.deviceIdCache.GetAttributesFromDevice(request.Context(), deviceId)
	if err != nil {
		c.logger.Error(err, "failed to get attributes from device", "deviceId", deviceId)
		return types.ConnectionResponse{
			Accept:         false,
			HTTPStatusCode: 404,
		}
	}
	serverOfferedResourceAttributes, err := calculateServerAttributes(k8sAttributes)
	if err != nil {
		c.logger.Error(err, "failed to calculate server attributes", "deviceId", deviceId)
		return types.ConnectionResponse{
			Accept:         false,
			HTTPStatusCode: 500,
		}
	}

	instrumentedAppName := workload.GetRuntimeObjectName(k8sAttributes.WorkloadName, k8sAttributes.WorkloadKind)

	return types.ConnectionResponse{
		ConnectionCallbacks: &ConnectionCallbacks{
			logger:                          c.logger,
			kubeclient:                      c.kubeclient,
			k8sAttributes:                   k8sAttributes,
			serverOfferedResourceAttributes: serverOfferedResourceAttributes,
			instrumentedAppName:             instrumentedAppName,
			pod:                             pod,
			scheme:                          c.scheme,
		},
		Accept:         true,
		HTTPStatusCode: 200,
	}
}

// ConnectionCallbacks is an implementation of the ConnectionCallbacks interface from opamp server.
var _ types.ConnectionCallbacks = &ConnectionCallbacks{}

type ConnectionCallbacks struct {
	logger                          logr.Logger
	scheme                          *runtime.Scheme // TODO: revisit this, we should not depend on controller runtime
	kubeclient                      client.Client
	k8sAttributes                   *deviceid.K8sResourceAttributes
	serverOfferedResourceAttributes []ResourceAttribute
	instrumentedAppName             string
	pod                             *corev1.Pod
	wasFirstMessageHandled          bool
}

func (c *ConnectionCallbacks) OnConnected(ctx context.Context, conn types.Connection) {
	println("OnConnected")
}

func (c *ConnectionCallbacks) OnMessage(ctx context.Context, conn types.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent {
	c.persistInstrumentationDeviceStatus(ctx, message)

	response := &protobufs.ServerToAgent{}
	if !c.wasFirstMessageHandled {
		c.OnFirstMessage(ctx, conn, message, response)
	}
	return response
}

func (c *ConnectionCallbacks) OnConnectionClose(conn types.Connection) {
	println("OnConnectionClose")
}

func (c *ConnectionCallbacks) persistInstrumentationDeviceStatus(ctx context.Context, message *protobufs.AgentToServer) error {
	if message.AgentDescription != nil {
		var pid int64
		for _, attr := range message.AgentDescription.IdentifyingAttributes {
			if attr.Key == string(semconv.AttributeProcessPID) {
				pid = attr.Value.GetIntValue()
				break
			}
		}
		if pid == 0 {
			return fmt.Errorf("missing pid in agent description")
		}

		identifyingAttributes := make([]odigosv1.Attribute, 0, len(message.AgentDescription.IdentifyingAttributes))
		for _, attr := range message.AgentDescription.IdentifyingAttributes {
			identifyingAttributes = append(identifyingAttributes, odigosv1.Attribute{
				Key:   attr.Key,
				Value: attr.GetValue().GetStringValue(),
			})
		}

		err := instrumentation_instance.PersistInstrumentationInstanceStatus(ctx, c.pod, c.kubeclient, c.instrumentedAppName, int(pid), c.scheme,
			instrumentation_instance.WithIdentifyingAttributes(identifyingAttributes),
			instrumentation_instance.WithMessage("Agent connected"),
		)
		if err != nil {
			return fmt.Errorf("failed to persist instrumentation instance status: %w", err)
		}
	}

	return nil
}

func (c *ConnectionCallbacks) OnFirstMessage(ctx context.Context, conn types.Connection, message *protobufs.AgentToServer, response *protobufs.ServerToAgent) {

	resourceAttributeConfig, err := json.Marshal(c.serverOfferedResourceAttributes)
	if err != nil {
		c.logger.Error(err, "failed to marshal server offered resource attributes")
		return
	}

	cfg := protobufs.AgentRemoteConfig{
		Config: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				"server-resolved-resource-attributes": {
					Body:        resourceAttributeConfig,
					ContentType: "application/json",
				},
			},
		},
	}

	response.RemoteConfig = &cfg
	c.wasFirstMessageHandled = true
}

func getDeviceIdFromHeader(request *http.Request) (string, error) {
	authorization := request.Header.Get("Authorization")
	if authorization == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	// make sure the Authorization header is in the format "DeviceId <device-id>"
	const prefix = "DeviceId "
	if len(authorization) <= len(prefix) || authorization[:len(prefix)] != prefix {
		return "", fmt.Errorf("authorization header is not in the format 'DeviceId <device-id>'")
	}

	return authorization[len(prefix):], nil
}
