package source

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
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

type ContainerRuntimeInfo struct {
	ContainerName  properties.EntityProperty   `json:"containerName"`
	Language       properties.EntityProperty   `json:"language"`
	RuntimeVersion properties.EntityProperty   `json:"runtimeVersion"`
	EnvVars        []properties.EntityProperty `json:"envVars"`
}

type RuntimeInfoAnalyze struct {
	Generation properties.EntityProperty `json:"generation"`
	Containers []ContainerRuntimeInfo    `json:"containers"`
}

type SourceAnalyze struct {
	Name      properties.EntityProperty    `json:"name"`
	Kind      properties.EntityProperty    `json:"kind"`
	Namespace properties.EntityProperty    `json:"namespace"`
	Labels    InstrumentationLabelsAnalyze `json:"labels"`

	InstrumentationConfig InstrumentationConfigAnalyze `json:"instrumentationConfig"`
	RuntimeInfo           *RuntimeInfoAnalyze          `json:"runtimeInfo"`
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

	// instrumentationConfigNotFound := instrumentationConfig == nil
	// statusAsExpected := instrumentationConfigNotFound == !instrumented
	// sb.WriteString("\nInstrumentation Config:\n")
	// if instrumentationConfigNotFound {
	// 	if statusAsExpected {
	// 		sb.WriteString(wrapTextInGreen("  Workload not instrumented, no instrumentation config\n"))
	// 	} else {
	// 		sb.WriteString("  Not yet created\n")
	// 	}
	// } else {
	// 	createAtText := "  Created at " + instrumentationConfig.GetCreationTimestamp().String()
	// 	sb.WriteString(wrapTextSuccessOfFailure(createAtText, statusAsExpected) + "\n")
	// }

	// if !statusAsExpected {
	// 	sb.WriteString("  Troubleshooting: https://docs.odigos.io/architecture/troubleshooting#2-odigos-instrumentation-config\n")
	// }

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

func analyzeRuntimeInfo(resources *OdigosSourceResources) *RuntimeInfoAnalyze {
	if resources.InstrumentationConfig == nil {
		return nil
	}

	generation := properties.EntityProperty{
		Name:  "Workload Generation",
		Value: resources.InstrumentationConfig.Status.ObservedWorkloadGeneration,
	}

	containers := make([]ContainerRuntimeInfo, 0, len(resources.InstrumentationConfig.Status.RuntimeDetailsByContainer))

	for _, container := range resources.InstrumentationConfig.Status.RuntimeDetailsByContainer {

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

		containers = append(containers, ContainerRuntimeInfo{
			ContainerName:  containerName,
			Language:       language,
			RuntimeVersion: runtimeVersion,
			EnvVars:        envVars,
		})
	}

	return &RuntimeInfoAnalyze{
		Generation: generation,
		Containers: containers,
	}
}

func AnalyzeSource(resources *OdigosSourceResources, workloadObj *K8sSourceObject) *SourceAnalyze {

	labelsAnalysis, instrumented := analyzeInstrumentationLabels(resources, workloadObj)
	icAnalysis := analyzeInstrumentationConfig(resources, instrumented)
	runtimeAnalysis := analyzeRuntimeInfo(resources, instrumented)

	return &SourceAnalyze{
		Name:      properties.EntityProperty{Name: "Name", Value: workloadObj.GetName()},
		Kind:      properties.EntityProperty{Name: "Kind", Value: workloadObj.Kind},
		Namespace: properties.EntityProperty{Name: "Namespace", Value: workloadObj.GetNamespace()},
		Labels:    labelsAnalysis,

		InstrumentationConfig: icAnalysis,
		RuntimeInfo:           runtimeAnalysis,
	}
}
