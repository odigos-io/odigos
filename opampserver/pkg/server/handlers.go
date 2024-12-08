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
	di "github.com/odigos-io/odigos/opampserver/pkg/deviceid"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConnectionHandlers struct {
	deviceIdCache *di.DeviceIdCache
	sdkConfig     *sdkconfig.SdkConfigManager
	logger        logr.Logger
	kubeclient    client.Client
	kubeClientSet *kubernetes.Clientset
	scheme        *runtime.Scheme // TODO: revisit this, we should not depend on controller runtime
	nodeName      string
}

type opampAgentAttributesKeys struct {
	ProgrammingLanguage string
	ContainerName       string
	PodName             string
	Namespace           string
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

	attrs := extractOpampAgentAttributes(firstMessage.AgentDescription)

	if attrs.ProgrammingLanguage == "" {
		return nil, nil, fmt.Errorf("missing programming language in agent description")
	}

	k8sAttributes, pod, err := c.resolveK8sAttributes(ctx, attrs, deviceId)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process k8s attributes: %w", err)
	}

	podWorkload := workload.PodWorkload{
		Namespace: k8sAttributes.Namespace,
		Kind:      workload.WorkloadKind(k8sAttributes.WorkloadKind),
		Name:      k8sAttributes.WorkloadName,
	}

	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(k8sAttributes.WorkloadName, k8sAttributes.WorkloadKind)
	instrumentationConfig, err := configsections.GetWorkloadInstrumentationConfig(ctx, c.kubeclient, instrumentedAppName, podWorkload.Namespace)
	if err != nil {
		c.logger.Error(err, "failed to get instrumentation config", "instrumentedAppName", instrumentedAppName, "namespace", podWorkload.Namespace)
		return nil, nil, err
	}

	serviceName := instrumentationConfig.Spec.ServiceName
	remoteResourceAttributes, err := configresolvers.CalculateServerAttributes(k8sAttributes, c.nodeName, serviceName)
	if err != nil {
		c.logger.Error(err, "failed to calculate server attributes", "k8sAttributes", k8sAttributes)
		return nil, nil, err
	}

	fullRemoteConfig, err := c.sdkConfig.GetFullConfig(ctx, remoteResourceAttributes, &podWorkload, instrumentedAppName, instrumentationConfig)
	if err != nil {
		c.logger.Error(err, "failed to get full config", "k8sAttributes", k8sAttributes)
		return nil, nil, err
	}
	c.logger.Info("new OpAMP client connected", "deviceId", deviceId, "namespace", k8sAttributes.Namespace, "podName", k8sAttributes.PodName, "instrumentedAppName", instrumentedAppName, "workloadKind", k8sAttributes.WorkloadKind, "workloadName", k8sAttributes.WorkloadName, "containerName", k8sAttributes.ContainerName)

	connectionInfo := &connection.ConnectionInfo{
		DeviceId:                 deviceId,
		Workload:                 podWorkload,
		Pod:                      pod,
		ContainerName:            k8sAttributes.ContainerName,
		Pid:                      pid,
		ProgrammingLanguage:      attrs.ProgrammingLanguage,
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

// resolveK8sAttributes resolves K8s resource attributes using either direct attributes from opamp agent or device cache
func (c *ConnectionHandlers) resolveK8sAttributes(ctx context.Context, attrs opampAgentAttributesKeys, deviceId string) (*di.K8sResourceAttributes, *corev1.Pod, error) {

	if attrs.hasRequiredAttributes() {
		return resolveFromDirectAttributes(ctx, attrs, c.kubeClientSet)
	}
	return c.deviceIdCache.GetAttributesFromDevice(ctx, deviceId)
}

func extractOpampAgentAttributes(agentDescription *protobufs.AgentDescription) opampAgentAttributesKeys {
	result := opampAgentAttributesKeys{}

	for _, attr := range agentDescription.IdentifyingAttributes {
		switch attr.Key {
		case string(semconv.TelemetrySDKLanguageKey):
			result.ProgrammingLanguage = attr.Value.GetStringValue()
		case string(semconv.K8SContainerNameKey):
			result.ContainerName = attr.Value.GetStringValue()
		case string(semconv.K8SPodNameKey):
			result.PodName = attr.Value.GetStringValue()
		case string(semconv.K8SNamespaceNameKey):
			result.Namespace = attr.Value.GetStringValue()
		}
	}

	return result
}

func (k opampAgentAttributesKeys) hasRequiredAttributes() bool {
	return k.ContainerName != "" && k.PodName != "" && k.Namespace != ""
}

func resolveFromDirectAttributes(ctx context.Context, attrs opampAgentAttributesKeys, kubeClient *kubernetes.Clientset) (*di.K8sResourceAttributes, *corev1.Pod, error) {

	pod, err := kubeClient.CoreV1().Pods(attrs.Namespace).Get(ctx, attrs.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	var workloadName string
	var workloadKind workload.WorkloadKind

	ownerRefs := pod.GetOwnerReferences()
	for _, ownerRef := range ownerRefs {
		workloadName, workloadKind, err = workload.GetWorkloadFromOwnerReference(ownerRef)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workload from owner reference: %w", err)
		}
	}

	k8sAttributes := &di.K8sResourceAttributes{
		Namespace:     attrs.Namespace,
		PodName:       attrs.PodName,
		ContainerName: attrs.ContainerName,
		WorkloadKind:  string(workloadKind),
		WorkloadName:  workloadName,
	}

	return k8sAttributes, pod, nil
}
