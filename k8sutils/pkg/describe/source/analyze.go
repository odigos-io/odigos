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

type SourceAnalyze struct {
	Name      properties.EntityProperty `json:"name"`
	Kind      properties.EntityProperty `json:"kind"`
	Namespace properties.EntityProperty `json:"namespace"`

	Labels InstrumentationLabelsAnalyze `json:"labels"`
}

func analyzeInstrumentationLabels(resource *OdigosSourceResources, workloadObj *K8sSourceObject) InstrumentationLabelsAnalyze {

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
	}
}

func AnalyzeSource(resources *OdigosSourceResources, workloadObj *K8sSourceObject) *SourceAnalyze {
	return &SourceAnalyze{
		Name:      properties.EntityProperty{Name: "Name", Value: workloadObj.GetName()},
		Kind:      properties.EntityProperty{Name: "Kind", Value: workloadObj.Kind},
		Namespace: properties.EntityProperty{Name: "Namespace", Value: workloadObj.GetNamespace()},
		Labels:    analyzeInstrumentationLabels(resources, workloadObj),
	}
}
