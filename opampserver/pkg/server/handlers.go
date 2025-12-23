package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
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

func (c *ConnectionHandlers) OnNewConnection(ctx context.Context, firstMessage *protobufs.AgentToServer) (*connection.ConnectionInfo, *protobufs.ServerToAgent, error) {

	if firstMessage.AgentDescription == nil {
		// first message must be agent description.
		// it is, however, possible that the OpAMP server restarted, and the agent is trying to reconnect.
		// in which case we send back flag and request full status update.
		c.logger.Info("Agent description is missing in the first OpAMP message, requesting full state update")
		serverToAgent := &protobufs.ServerToAgent{
			Flags: uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState),
		}
		return nil, serverToAgent, nil
	}

	var vpid int64
	for _, attr := range firstMessage.AgentDescription.IdentifyingAttributes {
		if attr.Key == string(semconv.ProcessPIDKey) || attr.Key == "process.vpid" {
			vpid = attr.Value.GetIntValue()
			break
		}
	}
	if vpid == 0 {
		return nil, nil, fmt.Errorf("missing container pid in agent description")
	}

	attrs, err := extractOpampAgentAttributes(firstMessage.AgentDescription)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract agent attributes: %w", err)
	}

	k8sAttributes, pod, err := resolveFromDirectAttributes(ctx, attrs, c.kubeClientSet)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to process k8s attributes: %w", err)
	}

	podWorkload := k8sconsts.PodWorkload{
		Namespace: k8sAttributes.Namespace,
		Kind:      k8sconsts.WorkloadKind(k8sAttributes.WorkloadKind),
		Name:      k8sAttributes.WorkloadName,
	}

	instrumentedAppName := workload.CalculateWorkloadRuntimeObjectName(k8sAttributes.WorkloadName, k8sAttributes.WorkloadKind)
	instrumentationConfig, err := configsections.GetWorkloadInstrumentationConfig(ctx, c.kubeclient, instrumentedAppName, podWorkload.Namespace)
	if err != nil {
		c.logger.Error(err, "failed to get instrumentation config", "instrumentedAppName", instrumentedAppName, "namespace", podWorkload.Namespace)
		return nil, nil, err
	}

	serviceName := instrumentationConfig.Spec.ServiceName
	if serviceName == "" {
		serviceName = k8sAttributes.WorkloadName
	}
	remoteResourceAttributes, err := configresolvers.CalculateServerAttributes(k8sAttributes, c.nodeName, serviceName)
	if err != nil {
		c.logger.Error(err, "failed to calculate server attributes", "k8sAttributes", k8sAttributes)
		return nil, nil, err
	}

	fullRemoteConfig, err := c.sdkConfig.GetFullConfig(ctx, remoteResourceAttributes, &podWorkload, instrumentedAppName, attrs.ProgrammingLanguage, instrumentationConfig, k8sAttributes.ContainerName)
	if err != nil {
		c.logger.Error(err, "failed to get full config", "k8sAttributes", k8sAttributes)
		return nil, nil, err
	}
	c.logger.Info("new OpAMP client connected", "namespace", k8sAttributes.Namespace, "podName", k8sAttributes.PodName, "instrumentedAppName", instrumentedAppName, "workloadKind", k8sAttributes.WorkloadKind, "workloadName", k8sAttributes.WorkloadName, "containerName", k8sAttributes.ContainerName, "otelServiceName", serviceName)

	connectionInfo := &connection.ConnectionInfo{
		Workload:                 podWorkload,
		Pod:                      pod,
		ContainerName:            k8sAttributes.ContainerName,
		Pid:                      vpid,
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
		c.logger.Info("missing remote config status in agent to server message, sending full remote config", "workload", connectionInfo.Workload)
		response.RemoteConfig = connectionInfo.AgentRemoteConfig
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

func (c *ConnectionHandlers) UpdateInstrumentationInstanceStatus(ctx context.Context, message *protobufs.AgentToServer, connectionInfo *connection.ConnectionInfo) error {

	isAgentDisconnect := message.AgentDisconnect != nil
	hasHealth := message.Health != nil
	// When agent disconnects, it need to report that it's health and disconnected
	// 1. In case disconnect with healthy status, we will delete the instrumentationInstnace CR
	// 2. Otherwise, we will keep the CR in unhealthy state so it can be used for troubleshooting
	if isAgentDisconnect {
		if !hasHealth {
			return fmt.Errorf("missing health in agent disconnect message")
		}
		// [1] - agent disconnects with healthy status, delete the instrumentation instance CR
		if message.Health.Healthy {
			c.logger.Info("Agent disconnected with healthy status, deleting instrumentation instance", "workloadNamespace", connectionInfo.Workload.Namespace, "workloadName", connectionInfo.Workload.Name, "workloadKind", connectionInfo.Workload.Kind)
			return instrumentation_instance.DeleteInstrumentationInstance(ctx, connectionInfo.Pod, connectionInfo.ContainerName, c.kubeclient, int(connectionInfo.Pid))
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

		if len(message.Health.ComponentHealthMap) > 0 {
			components := make([]odigosv1.InstrumentationLibraryStatus, 0, len(message.Health.ComponentHealthMap))
			for name, comp := range message.Health.ComponentHealthMap {
				libStatus := odigosv1.InstrumentationLibraryStatus{
					Name:           name,
					Type:           odigosv1.InstrumentationLibraryTypeInstrumentation,
					Healthy:        &comp.Healthy,
					LastStatusTime: metav1.Now(),
				}

				// Try parsing comp.Status JSON string into NonIdentifyingAttributes
				if comp.Status != "" {
					var parsed map[string]interface{}
					err := json.Unmarshal([]byte(comp.Status), &parsed)
					if err != nil {
						// fallback to setting it as plain message if parsing fails
						libStatus.Reason = "instrumentation.details"
						libStatus.Message = comp.Status
					} else {
						// set key-value pairs as NonIdentifyingAttributes
						attrs := make([]odigosv1.Attribute, 0, len(parsed))
						for k, v := range parsed {
							attrs = append(attrs, odigosv1.Attribute{
								Key:   k,
								Value: fmt.Sprintf("%v", v),
							})
						}
						libStatus.NonIdentifyingAttributes = attrs
					}
				}

				components = append(components, libStatus)
			}
			dynamicOptions = append(dynamicOptions, instrumentation_instance.WithComponents(components))
		}
	}

	if len(dynamicOptions) > 0 {
		err := instrumentation_instance.UpdateInstrumentationInstanceStatus(ctx, connectionInfo.Pod, connectionInfo.ContainerName, c.kubeclient, connectionInfo.InstrumentedAppName, int(connectionInfo.Pid), c.scheme, dynamicOptions...)
		if err != nil {
			return fmt.Errorf("failed to persist instrumentation instance status: %w", err)
		}
	}

	return nil
}

func extractOpampAgentAttributes(agentDescription *protobufs.AgentDescription) (opampAgentAttributesKeys, error) {
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

	if result.ProgrammingLanguage == "" {
		return result, fmt.Errorf("missing programming language in agent description")
	}
	if result.ContainerName == "" {
		return result, fmt.Errorf("missing container name in agent description")
	}
	if result.PodName == "" {
		return result, fmt.Errorf("missing pod name in agent description")
	}
	if result.Namespace == "" {
		return result, fmt.Errorf("missing namespace in agent description")
	}

	return result, nil
}

func resolveFromDirectAttributes(ctx context.Context, attrs opampAgentAttributesKeys, kubeClient *kubernetes.Clientset) (*configresolvers.K8sResourceAttributes, *corev1.Pod, error) {

	pod, err := kubeClient.CoreV1().Pods(attrs.Namespace).Get(ctx, attrs.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	var workloadName string
	var workloadKind k8sconsts.WorkloadKind

	ownerRefs := pod.GetOwnerReferences()
	for _, ownerRef := range ownerRefs {
		workloadName, workloadKind, err = workload.GetWorkloadFromOwnerReference(ownerRef, pod)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workload from owner reference: %w", err)
		}
	}

	k8sAttributes := &configresolvers.K8sResourceAttributes{
		Namespace:     attrs.Namespace,
		PodName:       attrs.PodName,
		ContainerName: attrs.ContainerName,
		WorkloadKind:  string(workloadKind),
		WorkloadName:  workloadName,
	}

	return k8sAttributes, pod, nil
}
