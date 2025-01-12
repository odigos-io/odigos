package source

import (
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
	"github.com/odigos-io/odigos/k8sutils/pkg/envoverwrite"
)

type InstrumentationLabelsAnalyze struct {
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

type RuntimeInfoAnalyze struct {
	Generation properties.EntityProperty     `json:"generation"`
	Containers []ContainerRuntimeInfoAnalyze `json:"containers"`
}

type InstrumentationConfigAnalyze struct {
	Created    properties.EntityProperty     `json:"created"`
	CreateTime *properties.EntityProperty    `json:"createTime"`
	Containers []ContainerRuntimeInfoAnalyze `json:"containers"`
}

type ContainerWorkloadManifestAnalyze struct {
	ContainerName properties.EntityProperty   `json:"containerName"`
	Devices       properties.EntityProperty   `json:"devices"`
	OriginalEnv   []properties.EntityProperty `json:"originalEnv"`
}

type InstrumentationDeviceAnalyze struct {
	StatusText properties.EntityProperty          `json:"statusText"`
	Containers []ContainerWorkloadManifestAnalyze `json:"containers"`
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
	Name      properties.EntityProperty    `json:"name"`
	Kind      properties.EntityProperty    `json:"kind"`
	Namespace properties.EntityProperty    `json:"namespace"`
	Labels    InstrumentationLabelsAnalyze `json:"labels"`

	RuntimeInfo           *RuntimeInfoAnalyze          `json:"runtimeInfo"`
	InstrumentationConfig InstrumentationConfigAnalyze `json:"instrumentationConfig"`
	InstrumentationDevice InstrumentationDeviceAnalyze `json:"instrumentationDevice"`

	TotalPods       int          `json:"totalPods"`
	PodsPhasesCount string       `json:"podsPhasesCount"`
	Pods            []PodAnalyze `json:"pods"`
}

func analyzeInstrumentationLabels(resource *OdigosSourceResources, workloadObj *K8sSourceObject) (InstrumentationLabelsAnalyze, bool) {
	workloadLabel, workloadFound := workloadObj.GetLabels()[consts.OdigosInstrumentationLabel]
	nsLabel, nsFound := resource.Namespace.GetLabels()[consts.OdigosInstrumentationLabel]

	workload := &properties.EntityProperty{Name: "Workload", Value: "unset",
		Explain: "the value of the odigos-instrumentation label on the workload object in k8s"}
	if workloadFound {
		workload.Value = fmt.Sprintf("%s=%s", consts.OdigosInstrumentationLabel, workloadLabel)
	}

	ns := &properties.EntityProperty{Name: "Namespace", Value: "unset",
		Explain: "the value of the odigos-instrumentation label on the namespace object in k8s"}
	if nsFound {
		ns.Value = fmt.Sprintf("%s=%s", consts.OdigosInstrumentationLabel, nsLabel)
	}

	var instrumented bool
	var decisionText string

	if workloadFound {
		instrumented = workloadLabel == consts.InstrumentationEnabled
		if instrumented {
			decisionText = "Workload is instrumented because the " + workloadObj.Kind + " contains the label '" +
				consts.OdigosInstrumentationLabel + "=" + workloadLabel + "'"
		} else {
			decisionText = "Workload is NOT instrumented because the " + workloadObj.Kind + " contains the label '" +
				consts.OdigosInstrumentationLabel + "=" + workloadLabel + "'"
		}
	} else {
		instrumented = nsLabel == consts.InstrumentationEnabled
		if instrumented {
			decisionText = "Workload is instrumented because the " + workloadObj.Kind +
				" is not labeled, and the namespace is labeled with '" + consts.OdigosInstrumentationLabel + "=" + nsLabel + "'"
		} else {
			if nsFound {
				decisionText = "Workload is NOT instrumented because the " + workloadObj.Kind +
					" is not labeled, and the namespace is labeled with '" + consts.OdigosInstrumentationLabel + "=" + nsLabel + "'"
			} else {
				decisionText = "Workload is NOT instrumented because neither the workload nor the namespace has the '" +
					consts.OdigosInstrumentationLabel + "' label set"
			}
		}
	}

	instrumentedProperty := properties.EntityProperty{
		Name:    "Instrumented",
		Value:   instrumented,
		Explain: "whether this workload is considered for instrumentation based on the presence of the odigos-instrumentation label",
	}
	decisionTextProperty := properties.EntityProperty{
		Name:    "DecisionText",
		Value:   decisionText,
		Explain: "a human readable explanation of the decision to instrument or not instrument this workload",
	}

	return InstrumentationLabelsAnalyze{
		Instrumented:     instrumentedProperty,
		Workload:         workload,
		Namespace:        ns,
		InstrumentedText: decisionTextProperty,
	}, instrumented
}

func analyzeInstrumentationConfig(resources *OdigosSourceResources, instrumented bool) InstrumentationConfigAnalyze {
	instrumentationConfigCreated := resources.InstrumentationConfig != nil

	created := properties.EntityProperty{
		Name:   "Created",
		Value:  properties.GetTextCreated(instrumentationConfigCreated),
		Status: properties.GetSuccessOrTransitioning(instrumentationConfigCreated == instrumented),
		Explain: "whether the instrumentation config object exists in the cluster. When a workload is labeled for instrumentation," +
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

	containers := make([]ContainerRuntimeInfoAnalyze, 0)
	if instrumentationConfigCreated {
		containers = analyzeRuntimeDetails(resources.InstrumentationConfig.Status.RuntimeDetailsByContainer)
	}

	return InstrumentationConfigAnalyze{
		Created:    created,
		CreateTime: createdTime,
		Containers: containers,
	}
}

func analyzeRuntimeDetails(runtimeDetailsByContainer []odigosv1.RuntimeDetailsByContainer) []ContainerRuntimeInfoAnalyze {
	containers := make([]ContainerRuntimeInfoAnalyze, 0, len(runtimeDetailsByContainer))

	for i := range runtimeDetailsByContainer {
		container := runtimeDetailsByContainer[i]
		containerName := properties.EntityProperty{
			Name:    "Container Name",
			Value:   container.ContainerName,
			Explain: "the unique name of the container in the k8s pod",
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

func analyzeInstrumentationDevice(resources *OdigosSourceResources, workloadObj *K8sSourceObject, instrumented bool) InstrumentationDeviceAnalyze {
	instrumentationConfig := resources.InstrumentationConfig

	appliedInstrumentationDeviceStatusMessage := "Unknown"
	var appliedDeviceStatus properties.PropertyStatus
	if !instrumented {
		// if the workload is not instrumented, the instrumentation device expected
		appliedInstrumentationDeviceStatusMessage = "No instrumentation devices expected"
		appliedDeviceStatus = properties.PropertyStatusSuccess
	}
	if instrumentationConfig != nil && instrumentationConfig.Status.Conditions != nil {
		for _, condition := range instrumentationConfig.Status.Conditions {
			if condition.Type == "AppliedInstrumentationDevice" { // TODO: share this constant with instrumentor
				if condition.ObservedGeneration == instrumentationConfig.GetGeneration() {
					appliedInstrumentationDeviceStatusMessage = condition.Message
					if condition.Status == metav1.ConditionTrue {
						appliedDeviceStatus = properties.PropertyStatusSuccess
					} else {
						appliedDeviceStatus = properties.PropertyStatusError
					}
				} else {
					appliedInstrumentationDeviceStatusMessage = "Waiting for reconciliation"
					appliedDeviceStatus = properties.PropertyStatusTransitioning
				}
				break
			}
		}
	}

	statusText := properties.EntityProperty{
		Name:    "Status",
		Value:   appliedInstrumentationDeviceStatusMessage,
		Status:  appliedDeviceStatus,
		Explain: "the result of applying the instrumentation device to the workload manifest",
	}

	// get original env vars:
	origWorkloadEnvValues, _ := envoverwrite.NewOrigWorkloadEnvValues(workloadObj.GetAnnotations())

	templateContainers := workloadObj.PodTemplateSpec.Spec.Containers
	containers := make([]ContainerWorkloadManifestAnalyze, 0, len(templateContainers))
	for i := range templateContainers {
		container := templateContainers[i]
		containerName := properties.EntityProperty{
			Name:    "Container Name",
			Value:   container.Name,
			Explain: "the unique name of the container in the k8s pod",
		}

		odigosDevices := make([]string, 0)
		for resourceName := range container.Resources.Limits {
			deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
			if found {
				odigosDevices = append(odigosDevices, deviceName)
			}
		}

		devices := properties.EntityProperty{
			Name:    "Devices",
			Value:   odigosDevices,
			Explain: "the odigos instrumentation devices that were added to the workload manifest",
		}

		originalContainerEnvs := origWorkloadEnvValues.GetContainerStoredEnvs(container.Name)
		originalEnv := make([]properties.EntityProperty, 0, len(originalContainerEnvs))
		for envName, envValue := range originalContainerEnvs {
			if envValue == nil {
				originalEnv = append(originalEnv, properties.EntityProperty{
					Name:    envName,
					Value:   "unset",
					Explain: "the original value of the environment variable in the workload manifest, before it was patched by odigos",
				})
			} else {
				originalEnv = append(originalEnv, properties.EntityProperty{
					Name:    envName,
					Value:   *envValue,
					Explain: "the original value of the environment variable in the workload manifest, before it was patched by odigos",
				})
			}
		}

		containers = append(containers, ContainerWorkloadManifestAnalyze{
			ContainerName: containerName,
			Devices:       devices,
			OriginalEnv:   originalEnv,
		})
	}

	return InstrumentationDeviceAnalyze{
		StatusText: statusText,
		Containers: containers,
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
		}
	} else {
		healthy = properties.EntityProperty{
			Name:    "Healthy",
			Value:   *instrumentationInstance.Status.Healthy,
			Status:  properties.GetSuccessOrError(*instrumentationInstance.Status.Healthy),
			Explain: "health indication for the instrumentation running for this process",
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

func analyzePods(resources *OdigosSourceResources, expectedDevices InstrumentationDeviceAnalyze) ([]PodAnalyze, string) {
	pods := make([]PodAnalyze, 0, len(resources.Pods.Items))
	podsStatuses := make(map[corev1.PodPhase]int)
	for i := range resources.Pods.Items {
		pod := resources.Pods.Items[i]
		podsStatuses[pod.Status.Phase]++

		name := properties.EntityProperty{
			Name:    "Pod Name",
			Value:   pod.GetName(),
			Explain: "the name of the k8s pod object that is part of the source workload",
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
			}

			deviceNames := make([]string, 0)
			for resourceName := range container.Resources.Limits {
				deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
				if found {
					deviceNames = append(deviceNames, deviceName)
				}
			}

			var expectedContainer *ContainerWorkloadManifestAnalyze
			for i := range expectedDevices.Containers {
				c := expectedDevices.Containers[i]
				if c.ContainerName.Value == container.Name {
					expectedContainer = &c
					break
				}
			}
			devicesStatus := properties.GetSuccessOrError(expectedContainer != nil && reflect.DeepEqual(deviceNames, expectedContainer.Devices.Value))
			actualDevices := properties.EntityProperty{
				Name:    "Actual Devices",
				Value:   deviceNames,
				Status:  devicesStatus,
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
	labelsAnalysis, instrumented := analyzeInstrumentationLabels(resources, workloadObj)
	runtimeAnalysis := analyzeRuntimeInfo(resources)
	icAnalysis := analyzeInstrumentationConfig(resources, instrumented)
	device := analyzeInstrumentationDevice(resources, workloadObj, instrumented)
	pods, podsText := analyzePods(resources, device)

	return &SourceAnalyze{
		Name: properties.EntityProperty{Name: "Name", Value: workloadObj.GetName(),
			Explain: "the name of the k8s workload object that this source describes"},
		Kind: properties.EntityProperty{Name: "Kind", Value: workloadObj.Kind,
			Explain: "the kind of the k8s workload object that this source describes (deployment/daemonset/statefulset)"},
		Namespace: properties.EntityProperty{Name: "Namespace", Value: workloadObj.GetNamespace(),
			Explain: "the namespace of the k8s workload object that this source describes"},
		Labels: labelsAnalysis,

		RuntimeInfo:           runtimeAnalysis,
		InstrumentationConfig: icAnalysis,
		InstrumentationDevice: device,

		TotalPods:       len(pods),
		PodsPhasesCount: podsText,
		Pods:            pods,
	}
}
