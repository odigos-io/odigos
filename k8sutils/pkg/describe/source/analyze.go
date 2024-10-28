package source

import (
	"fmt"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
	"github.com/odigos-io/odigos/k8sutils/pkg/envoverwrite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstrumentationLabelsAnalyze struct {
	Instrumented     properties.EntityProperty  `json:"instrumented"`
	Workload         *properties.EntityProperty `json:"workload"`
	Namespace        *properties.EntityProperty `json:"namespace"`
	InstrumentedText properties.EntityProperty  `json:"instrumentedText"`
}

type InstrumentationConfigAnalyze struct {
	Created    properties.EntityProperty  `json:"created"`
	CreateTime *properties.EntityProperty `json:"createTime"`
}

type ContainerRuntimeInfoAnalyze struct {
	ContainerName  properties.EntityProperty   `json:"containerName"`
	Language       properties.EntityProperty   `json:"language"`
	RuntimeVersion properties.EntityProperty   `json:"runtimeVersion"`
	EnvVars        []properties.EntityProperty `json:"envVars"`
}

type RuntimeInfoAnalyze struct {
	Generation properties.EntityProperty     `json:"generation"`
	Containers []ContainerRuntimeInfoAnalyze `json:"containers"`
}

type InstrumentedApplicationAnalyze struct {
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

type SourceAnalyze struct {
	Name      properties.EntityProperty    `json:"name"`
	Kind      properties.EntityProperty    `json:"kind"`
	Namespace properties.EntityProperty    `json:"namespace"`
	Labels    InstrumentationLabelsAnalyze `json:"labels"`

	InstrumentationConfig   InstrumentationConfigAnalyze   `json:"instrumentationConfig"`
	RuntimeInfo             *RuntimeInfoAnalyze            `json:"runtimeInfo"`
	InstrumentedApplication InstrumentedApplicationAnalyze `json:"instrumentedApplication"`
	InstrumentationDevice   InstrumentationDeviceAnalyze   `json:"instrumentationDevice"`
}

func analyzeInstrumentationLabels(resource *OdigosSourceResources, workloadObj *K8sSourceObject) (InstrumentationLabelsAnalyze, bool) {

	workloadLabel, workloadFound := workloadObj.GetLabels()[consts.OdigosInstrumentationLabel]
	nsLabel, nsFound := resource.Namespace.GetLabels()[consts.OdigosInstrumentationLabel]

	workload := &properties.EntityProperty{Name: "Workload", Value: "unset"}
	if workloadFound {
		workload.Value = fmt.Sprintf("%s=%s", consts.OdigosInstrumentationLabel, workloadLabel)
	}

	ns := &properties.EntityProperty{Name: "Namespace", Value: "unset"}
	if nsFound {
		ns.Value = fmt.Sprintf("%s=%s", consts.OdigosInstrumentationLabel, nsLabel)
	}

	var instrumented bool
	var decisionText string

	if workloadFound {
		instrumented = workloadLabel == consts.InstrumentationEnabled
		if instrumented {
			decisionText = "Workload is instrumented because the " + workloadObj.Kind + " contains the label '" + consts.OdigosInstrumentationLabel + "=" + workloadLabel + "'"
		} else {
			decisionText = "Workload is NOT instrumented because the " + workloadObj.Kind + " contains the label '" + consts.OdigosInstrumentationLabel + "=" + workloadLabel + "'"
		}
	} else {
		instrumented = nsLabel == consts.InstrumentationEnabled
		if instrumented {
			decisionText = "Workload is instrumented because the " + workloadObj.Kind + " is not labeled, and the namespace is labeled with '" + consts.OdigosInstrumentationLabel + "=" + nsLabel + "'"
		} else {
			if nsFound {
				decisionText = "Workload is NOT instrumented because the " + workloadObj.Kind + " is not labeled, and the namespace is labeled with '" + consts.OdigosInstrumentationLabel + "=" + nsLabel + "'"
			} else {
				decisionText = "Workload is NOT instrumented because neither the workload nor the namespace has the '" + consts.OdigosInstrumentationLabel + "' label set"
			}
		}
	}

	instrumentedProperty := properties.EntityProperty{
		Name:  "Instrumented",
		Value: instrumented,
	}
	decisionTextProperty := properties.EntityProperty{
		Name:  "DecisionText",
		Value: decisionText,
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
	}

	var createdTime *properties.EntityProperty
	if instrumentationConfigCreated {
		createdTime = &properties.EntityProperty{
			Name:  "create time",
			Value: resources.InstrumentationConfig.GetCreationTimestamp().String(),
		}
	}

	return InstrumentationConfigAnalyze{
		Created:    created,
		CreateTime: createdTime,
	}
}

func analyzeRuntimeDetails(runtimeDetailsByContainer []odigosv1.RuntimeDetailsByContainer) []ContainerRuntimeInfoAnalyze {
	containers := make([]ContainerRuntimeInfoAnalyze, 0, len(runtimeDetailsByContainer))

	for _, container := range runtimeDetailsByContainer {

		containerName := properties.EntityProperty{
			Name:  "Container Name",
			Value: container.ContainerName,
		}

		language := properties.EntityProperty{
			Name:   "Programming Language",
			Value:  container.Language,
			Status: properties.GetSuccessOrError(container.Language != common.UnknownProgrammingLanguage),
		}

		runtimeVersion := properties.EntityProperty{
			Name:  "Runtime Version",
			Value: container.RuntimeVersion,
		}
		if container.RuntimeVersion == "" {
			runtimeVersion.Value = "not available"
		}

		envVars := make([]properties.EntityProperty, 0, len(container.EnvVars))
		for _, envVar := range container.EnvVars {
			envVars = append(envVars, properties.EntityProperty{
				Name:  envVar.Name,
				Value: envVar.Value,
			})
		}

		containers = append(containers, ContainerRuntimeInfoAnalyze{
			ContainerName:  containerName,
			Language:       language,
			RuntimeVersion: runtimeVersion,
			EnvVars:        envVars,
		})
	}

	return containers
}

func analyzeRuntimeInfo(resources *OdigosSourceResources) *RuntimeInfoAnalyze {
	if resources.InstrumentationConfig == nil {
		return nil
	}

	generation := properties.EntityProperty{
		Name:  "Workload Generation",
		Value: resources.InstrumentationConfig.Status.ObservedWorkloadGeneration,
	}

	return &RuntimeInfoAnalyze{
		Generation: generation,
		Containers: analyzeRuntimeDetails(resources.InstrumentationConfig.Status.RuntimeDetailsByContainer),
	}
}

func analyzeInstrumentedApplication(resources *OdigosSourceResources) InstrumentedApplicationAnalyze {
	instrumentedApplicationCreated := resources.InstrumentedApplication != nil

	created := properties.EntityProperty{
		Name:   "Created",
		Value:  properties.GetTextCreated(instrumentedApplicationCreated),
		Status: properties.GetSuccessOrTransitioning(instrumentedApplicationCreated),
	}

	var createdTime *properties.EntityProperty
	if instrumentedApplicationCreated {
		createdTime = &properties.EntityProperty{
			Name:  "create time",
			Value: resources.InstrumentedApplication.GetCreationTimestamp().String(),
		}
	}

	return InstrumentedApplicationAnalyze{
		Created:    created,
		CreateTime: createdTime,
		Containers: analyzeRuntimeDetails(resources.InstrumentedApplication.Spec.RuntimeDetails),
	}
}

func analyzeInstrumentationDevice(resources *OdigosSourceResources, workloadObj *K8sSourceObject, instrumented bool) InstrumentationDeviceAnalyze {

	instrumentedApplication := resources.InstrumentedApplication

	appliedInstrumentationDeviceStatusMessage := "Unknown"
	var appliedDeviceStatus properties.PropertyStatus
	if !instrumented {
		// if the workload is not instrumented, the instrumentation device expected
		appliedInstrumentationDeviceStatusMessage = "No instrumentation devices expected"
		appliedDeviceStatus = properties.PropertyStatusSuccess
	}
	if instrumentedApplication != nil && instrumentedApplication.Status.Conditions != nil {
		for _, condition := range instrumentedApplication.Status.Conditions {
			if condition.Type == "AppliedInstrumentationDevice" { // TODO: share this constant with instrumentor
				if condition.ObservedGeneration == instrumentedApplication.GetGeneration() {
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
		Name:   "Status",
		Value:  appliedInstrumentationDeviceStatusMessage,
		Status: appliedDeviceStatus,
	}

	// get original env vars:
	origWorkloadEnvValues, _ := envoverwrite.NewOrigWorkloadEnvValues(workloadObj.GetAnnotations())

	templateContainers := workloadObj.PodTemplateSpec.Spec.Containers
	containers := make([]ContainerWorkloadManifestAnalyze, 0, len(templateContainers))
	for _, container := range templateContainers {

		containerName := properties.EntityProperty{
			Name:  "Container Name",
			Value: container.Name,
		}

		odigosDevices := make([]string, 0)
		for resourceName := range container.Resources.Limits {
			deviceName, found := strings.CutPrefix(resourceName.String(), common.OdigosResourceNamespace+"/")
			if found {
				odigosDevices = append(odigosDevices, deviceName)
			}
		}

		var devicesNames interface{}
		switch len(odigosDevices) {
		case 0:
			devicesNames = "No instrumentation devices"
		case 1:
			devicesNames = odigosDevices[0]
		default:
			devicesNames = odigosDevices
		}
		devices := properties.EntityProperty{
			Name:  "Devices",
			Value: devicesNames,
		}

		originalContainerEnvs := origWorkloadEnvValues.GetContainerStoredEnvs(container.Name)
		originalEnv := make([]properties.EntityProperty, 0, len(originalContainerEnvs))
		for envName, envValue := range originalContainerEnvs {
			if envValue == nil {
				originalEnv = append(originalEnv, properties.EntityProperty{
					Name:  envName,
					Value: "unset",
				})
			} else {
				originalEnv = append(originalEnv, properties.EntityProperty{
					Name:  envName,
					Value: *envValue,
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

func AnalyzeSource(resources *OdigosSourceResources, workloadObj *K8sSourceObject) *SourceAnalyze {

	labelsAnalysis, instrumented := analyzeInstrumentationLabels(resources, workloadObj)
	icAnalysis := analyzeInstrumentationConfig(resources, instrumented)
	runtimeAnalysis := analyzeRuntimeInfo(resources)
	instrumentedApplication := analyzeInstrumentedApplication(resources)
	device := analyzeInstrumentationDevice(resources, workloadObj, instrumented)

	return &SourceAnalyze{
		Name:      properties.EntityProperty{Name: "Name", Value: workloadObj.GetName()},
		Kind:      properties.EntityProperty{Name: "Kind", Value: workloadObj.Kind},
		Namespace: properties.EntityProperty{Name: "Namespace", Value: workloadObj.GetNamespace()},
		Labels:    labelsAnalysis,

		InstrumentationConfig:   icAnalysis,
		RuntimeInfo:             runtimeAnalysis,
		InstrumentedApplication: instrumentedApplication,
		InstrumentationDevice:   device,
	}
}
