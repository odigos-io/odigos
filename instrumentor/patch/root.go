package patch

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
)

const (
	NodeIPEnvName  = "NODE_IP"
	HostIPEnvValue = "$(NODE_IP)"
)

type CollectorInfo struct {
	Hostname string
	Port     int
}

type Patcher interface {
	Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication)
	Revert(podSpec *v1.PodTemplateSpec)
}

var patcherMap = map[common.ProgrammingLanguage]Patcher{
	common.JavaProgrammingLanguage:       java,
	common.PythonProgrammingLanguage:     python,
	common.DotNetProgrammingLanguage:     dotNet,
	common.JavascriptProgrammingLanguage: nodeJs,
	common.GoProgrammingLanguage:         golang,
}

func ModifyObject(original *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) error {
	for _, l := range getLangsInResult(instrumentation) {
		p, exists := patcherMap[l]
		if !exists {
			return fmt.Errorf("unable to find patcher for lang %s", l)
		}

		p.Patch(original, instrumentation)
	}

	return nil
}

func Revert(original *v1.PodTemplateSpec) {
	for _, p := range patcherMap {
		p.Revert(original)
	}
}

func removeDeviceFromPodSpec(deviceName v1.ResourceName, podSpec *v1.PodTemplateSpec) {
	for _, container := range podSpec.Spec.Containers {
		delete(container.Resources.Limits, deviceName)
		delete(container.Resources.Requests, deviceName)
	}
}

func getLangsInResult(instrumentation *odigosv1.InstrumentedApplication) []common.ProgrammingLanguage {
	langMap := make(map[common.ProgrammingLanguage]interface{})
	for _, c := range instrumentation.Spec.Languages {
		langMap[c.Language] = nil
	}

	var langs []common.ProgrammingLanguage
	for l, _ := range langMap {
		langs = append(langs, l)
	}

	return langs
}

func shouldPatch(instrumentation *odigosv1.InstrumentedApplication, lang common.ProgrammingLanguage, containerName string) bool {
	for _, l := range instrumentation.Spec.Languages {
		if l.ContainerName == containerName && l.Language == lang {
			// TODO: Handle CGO
			return true
		}
	}

	return false
}
