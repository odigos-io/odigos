package patch

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/instrumentor/api/v1"
	v1 "k8s.io/api/core/v1"
)

type CollectorInfo struct {
	Hostname string
	Port     int
}

type Patcher interface {
	Patch(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication)
}

var patcherMap = map[odigosv1.ProgrammingLanguage]Patcher{
	odigosv1.JavaProgrammingLanguage:       java,
	odigosv1.PythonProgrammingLanguage:     python,
	odigosv1.DotNetProgrammingLanguage:     dotNet,
	odigosv1.JavascriptProgrammingLanguage: nodeJs,
	odigosv1.GoProgrammingLanguage:         golang,
}

func AccordingToInstrumentationApp(podSpec *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication) error {
	for _, l := range getLangsInResult(instrumentation) {
		p, exists := patcherMap[l]
		if !exists {
			return fmt.Errorf("unable to find patcher for lang %s", l)
		}

		p.Patch(podSpec, instrumentation)

	}

	return nil
}

func getLangsInResult(instrumentation *odigosv1.InstrumentedApplication) []odigosv1.ProgrammingLanguage {
	langMap := make(map[odigosv1.ProgrammingLanguage]interface{})
	for _, c := range instrumentation.Spec.Languages {
		langMap[c.Language] = nil
	}

	var langs []odigosv1.ProgrammingLanguage
	for l, _ := range langMap {
		langs = append(langs, l)
	}

	return langs
}

func shouldPatch(instrumentation *odigosv1.InstrumentedApplication, lang odigosv1.ProgrammingLanguage, containerName string) bool {
	for _, l := range instrumentation.Spec.Languages {
		if l.ContainerName == containerName && l.Language == lang {
			// TODO: Handle CGO
			return true
		}
	}

	return false
}

func getIndexOfEnv(envs []v1.EnvVar, name string) int {
	for i := range envs {
		if envs[i].Name == name {
			return i
		}
	}
	return -1
}

func calculateAppName(podSpace *v1.PodTemplateSpec, currentContainer *v1.Container, instrumentation *odigosv1.InstrumentedApplication) string {
	if len(podSpace.Spec.Containers) > 1 {
		return currentContainer.Name
	}

	return instrumentation.Spec.Ref.Name
}
