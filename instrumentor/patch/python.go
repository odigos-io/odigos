package patch

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	pythonAgentName         = "edenfed/otel-python-agent:v0.2"
	pythonVolumeName        = "agentdir-python"
	pythonMountPath         = "/otel-auto-instrumentation"
	envOtelTracesExporter   = "OTEL_TRACES_EXPORTER"
	envOtelMetricsExporter  = "OTEL_METRICS_EXPORTER"
	envValOtelHttpExporter  = "otlp_proto_http"
	envLogCorrelation       = "OTEL_PYTHON_LOG_CORRELATION"
	pythonInitContainerName = "copy-python-agent"
)

var python = &pythonPatcher{}

type pythonPatcher struct{}

func (p *pythonPatcher) Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) {
	var modifiedContainers []v1.Container
	for _, container := range podSpec.Spec.Containers {
		if shouldPatch(instrumentation, common.PythonProgrammingLanguage, container.Name) {
			container.Resources.Limits["instrumentation.odigos.io/python"] = resource.MustParse("1")
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	podSpec.Spec.Containers = modifiedContainers
}

func (p *pythonPatcher) IsInstrumented(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) bool {
	for _, c := range podSpec.Spec.Containers {
		if _, exists := c.Resources.Limits["instrumentation.odigos.io/python"]; exists {
			return true
		}
	}
	return false
}
