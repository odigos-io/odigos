package source

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
)

type InstrumentationSourcesAnalyze struct {
	Instrumented     properties.EntityProperty  `json:"instrumented"`
	Workload         *properties.EntityProperty `json:"workload"`
	Namespace        *properties.EntityProperty `json:"namespace"`
	InstrumentedText properties.EntityProperty  `json:"instrumentedText"`
}

type ContainerRuntimeInfoAnalyze struct {
	ContainerName        properties.EntityProperty   `json:"containerName"`
	Language             properties.EntityProperty   `json:"language"`
	RuntimeVersion       properties.EntityProperty   `json:"runtimeVersion"`
	CriError             properties.EntityProperty   `json:"criError"`
	EnvVars              []properties.EntityProperty `json:"envVars"`
	ContainerRuntimeEnvs []properties.EntityProperty `json:"containerRuntimeEnvs"`
}

type ContainerAgentConfigAnalyze struct {
	ContainerName  properties.EntityProperty  `json:"containerName"`
	Instrumented   properties.EntityProperty  `json:"instrumented"`
	Reason         *properties.EntityProperty `json:"reason"`
	Message        *properties.EntityProperty `json:"message"`
	OtelDistroName *properties.EntityProperty `json:"otelDistroName"`
}

type RuntimeInfoAnalyze struct {
	Containers []ContainerRuntimeInfoAnalyze `json:"containers"`
}

type OtelAgentsAnalyze struct {
	Created    properties.EntityProperty     `json:"created"`
	CreateTime *properties.EntityProperty    `json:"createTime"`
	Containers []ContainerAgentConfigAnalyze `json:"containers"`
}

type InstrumentationInstanceAnalyze struct {
	Healthy               properties.EntityProperty   `json:"healthy"`
	Message               *properties.EntityProperty  `json:"message"`
	IdentifyingAttributes []properties.EntityProperty `json:"identifyingAttributes"`
}

type PodContainerAnalyze struct {
	ContainerName            properties.EntityProperty        `json:"containerName"`
	ActualDevices            properties.EntityProperty        `json:"actualDevices"`
	InstrumentationInstances []InstrumentationInstanceAnalyze `json:"instrumentationInstances"`
}

type PodAnalyze struct {
	PodName    properties.EntityProperty `json:"podName"`
	NodeName   properties.EntityProperty `json:"nodeName"`
	Phase      properties.EntityProperty `json:"phase"`
	Containers []PodContainerAnalyze     `json:"containers"`
}

type SourceAnalyze struct {
	Name                  properties.EntityProperty     `json:"name"`
	Kind                  properties.EntityProperty     `json:"kind"`
	Namespace             properties.EntityProperty     `json:"namespace"`
	SourceObjectsAnalysis InstrumentationSourcesAnalyze `json:"sourceObjects"`

	RuntimeInfo *RuntimeInfoAnalyze `json:"runtimeInfo"`
	OtelAgents  OtelAgentsAnalyze   `json:"otelAgents"`

	TotalPods       int          `json:"totalPods"`
	PodsPhasesCount string       `json:"podsPhasesCount"`
	Pods            []PodAnalyze `json:"pods"`
}

// Deprecated: Sources are used to mark workloads for instrumentation.
func analyzeInstrumentationBySources(sources *odigosv1.WorkloadSources) (InstrumentationSourcesAnalyze, bool) {
	workloadSource := sources.Workload
	nsSource := sources.Namespace

	workload := &properties.EntityProperty{Name: "Workload", Value: "unset",
		Explain: "existence of workload specific Source object in k8s"}
	if sources.Workload != nil && !sources.Workload.Spec.DisableInstrumentation {
		workload.Value = "instrumented"
	}

	ns := &properties.EntityProperty{Name: "Namespace", Value: "unset",
		Explain: "existence of namespace Source for this workload in k8s"}
	if sources.Namespace != nil && !sources.Namespace.Spec.DisableInstrumentation {
		ns.Value = "instrumented"
	}

	var instrumented bool
	var decisionText string

	if workloadSource != nil {
		instrumented = !workloadSource.Spec.DisableInstrumentation
		if instrumented {
			decisionText = "Workload is instrumented because the workload source is present and enabled"
		} else {
			decisionText = "Workload is NOT instrumented because the workload source is present and disabled"
		}
	} else {
		if nsSource != nil {
			instrumented = !nsSource.Spec.DisableInstrumentation
			if instrumented {
				decisionText = "Workload is instrumented because the workload source is not present, but the namespace source is present and enabled"
			} else {
				decisionText = "Workload is NOT instrumented because the workload source is not present, but the namespace source is present and disabled"
			}
		} else {
			instrumented = false
			decisionText = "Workload is NOT instrumented because neither the workload source nor the namespace source are present"
		}
	}

	instrumentedProperty := properties.EntityProperty{
		Name:    "Instrumented",
		Value:   instrumented,
		Explain: "whether this workload is considered for instrumentation based on the presence of the Source objects",
	}
	decisionTextProperty := properties.EntityProperty{
		Name:    "DecisionText",
		Value:   decisionText,
		Explain: "a human readable explanation of the decision to instrument or not instrument this workload",
	}

	return InstrumentationSourcesAnalyze{
		Instrumented:     instrumentedProperty,
		Workload:         workload,
		Namespace:        ns,
		InstrumentedText: decisionTextProperty,
	}, instrumented
}

func analyzeEnabledAgents(resources *OdigosSourceResources, instrumented bool) OtelAgentsAnalyze {
	instrumentationConfigCreated := resources.InstrumentationConfig != nil

	created := properties.EntityProperty{
		Name:   "Created",
		Value:  properties.GetTextCreated(instrumentationConfigCreated),
		Status: properties.GetSuccessOrTransitioning(instrumentationConfigCreated == instrumented),
		Explain: "whether the instrumentation config object exists in the cluster. When a Source object is created," +
			" an instrumentation config object is created",
	}

	var createdTime *properties.EntityProperty
	if instrumentationConfigCreated {
		createdTime = &properties.EntityProperty{
			Name:    "create time",
			Value:   resources.InstrumentationConfig.GetCreationTimestamp().String(),
			Explain: "the time when the instrumentation config object was created",
		}
	}

	containers := make([]ContainerAgentConfigAnalyze, 0)
	if instrumentationConfigCreated {
		containers = analyzeContainersConfig(&resources.InstrumentationConfig.Spec.Containers)
	}

	return OtelAgentsAnalyze{
		Created:    created,
		CreateTime: createdTime,
		Containers: containers,
	}
}

func analyzeContainersConfig(containers *[]odigosv1.ContainerConfig) []ContainerAgentConfigAnalyze {
	containersAnalysis := make([]ContainerAgentConfigAnalyze, 0, len(*containers))
	for i := range *containers {
		container := (*containers)[i]

		containerName := properties.EntityProperty{
			Name:    "Container Name",
			Value:   container.ContainerName,
			Explain: "the unique name of the container in the k8s pod",
			ListKey: true,
		}

		instrumented := properties.EntityProperty{
			Name:    "Agent Injection Enabled",
			Value:   container.Instrumented,
			Explain: "whether the container is instrumented by the odigos agent",
		}

		var reason *properties.EntityProperty
		if container.InstrumentationReason != "" {
			reason = &properties.EntityProperty{
				Name:    "Instrumentation Reason",
				Value:   string(container.InstrumentationReason),
				Explain: "the reason why the container is or is not instrumented by the odigos agent",
			}
		}

		var message *properties.EntityProperty
		if container.InstrumentationMessage != "" {
			message = &properties.EntityProperty{
				Name:    "Message",
				Value:   container.InstrumentationMessage,
				Explain: "a human readable message from the odigos agent indicating the status of the instrumentation",
			}
		}

		var otelDistroName *properties.EntityProperty
		if container.OtelDistroName != "" {
			otelDistroName = &properties.EntityProperty{
				Name:    "Otel Distro Name",
				Value:   container.OtelDistroName,
				Explain: "the name of the OpenTelemetry distribution that is being used to instrument this container",
			}
		}

		containersAnalysis = append(containersAnalysis, ContainerAgentConfigAnalyze{
			ContainerName:  containerName,
			Instrumented:   instrumented,
			Reason:         reason,
			Message:        message,
			OtelDistroName: otelDistroName,
		})
	}

	return containersAnalysis
}

func analyzeRuntimeDetails(runtimeDetailsByContainer []odigosv1.RuntimeDetailsByContainer) []ContainerRuntimeInfoAnalyze {
	containers := make([]ContainerRuntimeInfoAnalyze, 0, len(runtimeDetailsByContainer))

	for i := range runtimeDetailsByContainer {
		container := runtimeDetailsByContainer[i]
		containerName := properties.EntityProperty{
			Name:    "Container Name",
			Value:   container.ContainerName,
			Explain: "the unique name of the container in the k8s pod",
			ListKey: true,
		}

		language := properties.EntityProperty{
			Name:    "Programming Language",
			Value:   container.Language,
			Status:  properties.GetSuccessOrError(container.Language != common.UnknownProgrammingLanguage),
			Explain: "the programming language detected by odigos to be running in this container",
		}

		runtimeVersion := properties.EntityProperty{
			Name:    "Runtime Version",
			Value:   container.RuntimeVersion,
			Explain: "the version of the runtime detected by odigos to be running in this container",
		}
		if container.RuntimeVersion == "" {
			runtimeVersion.Value = "not available"
		}

		criError := properties.EntityProperty{
			Name:    "CRI Error",
			Explain: "an error message from the container runtime interface (CRI) when trying to get runtime details for this container",
		}
		if container.CriErrorMessage != nil {
			criError.Value = *container.CriErrorMessage
			criError.Status = properties.PropertyStatusError
		} else {
			criError.Value = "No CRI error observed"
		}

		envVars := make([]properties.EntityProperty, 0, len(container.EnvVars))
		for _, envVar := range container.EnvVars {
			envVars = append(envVars, properties.EntityProperty{
				Name:  envVar.Name,
				Value: envVar.Value,
			})
		}
		containerRuntimeEnvs := make([]properties.EntityProperty, 0, len(container.EnvFromContainerRuntime))
		for _, envVar := range container.EnvFromContainerRuntime {
			containerRuntimeEnvs = append(containerRuntimeEnvs, properties.EntityProperty{
				Name:  envVar.Name,
				Value: envVar.Value,
			})
		}

		containers = append(containers, ContainerRuntimeInfoAnalyze{
			ContainerName:        containerName,
			Language:             language,
			RuntimeVersion:       runtimeVersion,
			EnvVars:              envVars,
			ContainerRuntimeEnvs: containerRuntimeEnvs,
			CriError:             criError,
		})
	}

	return containers
}

func analyzeRuntimeInfo(resources *OdigosSourceResources) *RuntimeInfoAnalyze {
	if resources.InstrumentationConfig == nil {
		return nil
	}

	return &RuntimeInfoAnalyze{
		Containers: analyzeRuntimeDetails(resources.InstrumentationConfig.Status.RuntimeDetailsByContainer),
	}
}

func analyzeInstrumentationInstance(instrumentationInstance *odigosv1.InstrumentationInstance) InstrumentationInstanceAnalyze {
	var healthy properties.EntityProperty
	if instrumentationInstance.Status.Healthy == nil {
		healthy = properties.EntityProperty{
			Name:    "Healthy",
			Value:   "Not Reported",
			Status:  properties.PropertyStatusTransitioning,
			Explain: "health indication for the instrumentation running for this process",
			ListKey: true,
		}
	} else {
		healthy = properties.EntityProperty{
			Name:    "Healthy",
			Value:   *instrumentationInstance.Status.Healthy,
			Status:  properties.GetSuccessOrError(*instrumentationInstance.Status.Healthy),
			Explain: "health indication for the instrumentation running for this process",
			ListKey: true,
		}
	}

	var message *properties.EntityProperty
	if instrumentationInstance.Status.Message != "" {
		message = &properties.EntityProperty{
			Name:    "Message",
			Value:   instrumentationInstance.Status.Message,
			Explain: "a human readable message from the instrumentation indicating the health of the instrumentation running for this process",
		}
	}

	identifyingAttributes := make([]properties.EntityProperty, 0, len(instrumentationInstance.Status.IdentifyingAttributes))
	for _, attribute := range instrumentationInstance.Status.IdentifyingAttributes {
		identifyingAttributes = append(identifyingAttributes, properties.EntityProperty{
			Name:  attribute.Key,
			Value: attribute.Value,
		})
	}

	return InstrumentationInstanceAnalyze{
		Healthy:               healthy,
		Message:               message,
		IdentifyingAttributes: identifyingAttributes,
	}
}

func podPhaseToStatus(phase corev1.PodPhase) properties.PropertyStatus {
	switch phase {
	case corev1.PodSucceeded, corev1.PodRunning:
		return properties.PropertyStatusSuccess
	case corev1.PodPending:
		return properties.PropertyStatusTransitioning
	case corev1.PodFailed:
		return properties.PropertyStatusError
	default:
		return properties.PropertyStatusError
	}
}

func analyzePods(resources *OdigosSourceResources) ([]PodAnalyze, string) {
	pods := make([]PodAnalyze, 0, len(resources.Pods.Items))
	podsStatuses := make(map[corev1.PodPhase]int)
	for i := range resources.Pods.Items {
		pod := resources.Pods.Items[i]
		podsStatuses[pod.Status.Phase]++

		name := properties.EntityProperty{
			Name:    "Pod Name",
			Value:   pod.GetName(),
			Explain: "the name of the k8s pod object that is part of the source workload",
			ListKey: true,
		}
		nodeName := properties.EntityProperty{
			Name:    "Node Name",
			Value:   pod.Spec.NodeName,
			Explain: "the name of the k8s node where the current pod being described is scheduled",
		}

		var phase properties.EntityProperty

		if pod.DeletionTimestamp != nil {
			phase = properties.EntityProperty{
				Name:    "Phase",
				Value:   "Terminating",
				Status:  properties.PropertyStatusTransitioning,
				Explain: "the current pod phase for the pod being described",
			}
		} else {
			phase = properties.EntityProperty{
				Name:    "Phase",
				Value:   pod.Status.Phase,
				Status:  podPhaseToStatus(pod.Status.Phase),
				Explain: "the current pod phase for the pod being described",
			}
		}

		containers := make([]PodContainerAnalyze, 0, len(pod.Spec.Containers))
		for i := range pod.Spec.Containers {
			container := pod.Spec.Containers[i]
			containerName := properties.EntityProperty{
				Name:    "Container Name",
				Value:   container.Name,
				Explain: "the unique name of a container being described in the pod",
				ListKey: true,
			}

			deviceNames := make([]string, 0)
			for resourceName := range container.Resources.Limits {
				deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
				if found {
					deviceNames = append(deviceNames, deviceName)
				}
			}

			actualDevices := properties.EntityProperty{
				Name:    "Actual Devices",
				Value:   deviceNames,
				Explain: "the odigos instrumentation devices that were found on this pod container instance",
			}

			// find the instrumentation instances for this pod
			thisPodInstrumentationInstances := make([]InstrumentationInstanceAnalyze, 0)
			for i := range resources.InstrumentationInstances.Items {
				instance := resources.InstrumentationInstances.Items[i]
				if len(instance.OwnerReferences) != 1 || instance.OwnerReferences[0].Kind != "Pod" {
					continue
				}
				if instance.OwnerReferences[0].Name != pod.GetName() {
					continue
				}
				if instance.Spec.ContainerName != container.Name {
					continue
				}
				instanceAnalyze := analyzeInstrumentationInstance(&instance)
				thisPodInstrumentationInstances = append(thisPodInstrumentationInstances, instanceAnalyze)
			}

			containers = append(containers, PodContainerAnalyze{
				ContainerName:            containerName,
				ActualDevices:            actualDevices,
				InstrumentationInstances: thisPodInstrumentationInstances,
			})
		}

		pods = append(pods, PodAnalyze{
			PodName:    name,
			NodeName:   nodeName,
			Phase:      phase,
			Containers: containers,
		})
	}

	podPhasesTexts := make([]string, 0)
	for phase, count := range podsStatuses {
		podPhasesTexts = append(podPhasesTexts, fmt.Sprintf("%s %d", phase, count))
	}
	podPhasesText := strings.Join(podPhasesTexts, ", ")

	return pods, podPhasesText
}

func AnalyzeSource(resources *OdigosSourceResources, workloadObj *K8sSourceObject) *SourceAnalyze {
	sourcesAnalysis, instrumented := analyzeInstrumentationBySources(resources.Sources)
	runtimeAnalysis := analyzeRuntimeInfo(resources)
	icAnalysis := analyzeEnabledAgents(resources, instrumented)
	pods, podsText := analyzePods(resources)

	return &SourceAnalyze{
		Name: properties.EntityProperty{Name: "Name", Value: workloadObj.GetName(),
			Explain: "the name of the k8s workload object that this source describes"},
		Kind: properties.EntityProperty{Name: "Kind", Value: workloadObj.Kind,
			Explain: "the kind of the k8s workload object that this source describes (deployment/daemonset/statefulset)"},
		Namespace: properties.EntityProperty{Name: "Namespace", Value: workloadObj.GetNamespace(),
			Explain: "the namespace of the k8s workload object that this source describes"},
		SourceObjectsAnalysis: sourcesAnalysis,

		RuntimeInfo: runtimeAnalysis,
		OtelAgents:  icAnalysis,

		TotalPods:       len(pods),
		PodsPhasesCount: podsText,
		Pods:            pods,
	}
}
