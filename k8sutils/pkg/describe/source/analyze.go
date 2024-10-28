package source

import (
	"fmt"

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

type SourceAnalyze struct {
	Name      properties.EntityProperty    `json:"name"`
	Kind      properties.EntityProperty    `json:"kind"`
	Namespace properties.EntityProperty    `json:"namespace"`
	Labels    InstrumentationLabelsAnalyze `json:"labels"`

	InstrumentationConfig InstrumentationConfigAnalyze `json:"instrumentationConfig"`
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

func AnalyzeSource(resources *OdigosSourceResources, workloadObj *K8sSourceObject) *SourceAnalyze {

	labelsAnalysis, instrumented := analyzeInstrumentationLabels(resources, workloadObj)
	icAnalysis := analyzeInstrumentationConfig(resources, instrumented)

	return &SourceAnalyze{
		Name:      properties.EntityProperty{Name: "Name", Value: workloadObj.GetName()},
		Kind:      properties.EntityProperty{Name: "Kind", Value: workloadObj.Kind},
		Namespace: properties.EntityProperty{Name: "Namespace", Value: workloadObj.GetNamespace()},
		Labels:    labelsAnalysis,

		InstrumentationConfig: icAnalysis,
	}
}
