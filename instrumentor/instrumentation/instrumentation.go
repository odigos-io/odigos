package instrumentation

import (
	"errors"
	"strings"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	ErrNoDefaultSDK = errors.New("no default sdks found")
)

func ConfigureInstrumentationForPod(original *corev1.PodTemplateSpec, runtimeDetails []odigosv1.RuntimeDetailsByContainer, targetObj client.Object,
	logger logr.Logger, agentsCanRunConcurrently bool) (bool, error) {
	// delete any existing instrumentation devices.
	// this is necessary for example when migrating from community to enterprise,
	// and we need to cleanup the community device before adding the enterprise one.
	RevertInstrumentationDevices(original)

	deviceSkippedDueToOtherAgent := false
	var modifiedContainers []corev1.Container

	for _, container := range original.Spec.Containers {
		containerLanguage := getLanguageOfContainer(runtimeDetails, container.Name)
		containerHaveOtherAgent := getContainerOtherAgents(runtimeDetails, container.Name)

		// By default, Odigos does not run alongside other agents.
		// However, if configured in the odigos-config, it can be allowed to run in parallel.
		if containerHaveOtherAgent != nil && !agentsCanRunConcurrently {
			logger.Info("Container is running other agent, skip applying instrumentation device", "agent", containerHaveOtherAgent.Name, "container", container.Name)

			// Not actually modifying the container, but we need to append it to the list.
			modifiedContainers = append(modifiedContainers, container)
			deviceSkippedDueToOtherAgent = true
			continue
		}

		if containerLanguage == common.UnknownProgrammingLanguage ||
			containerLanguage == common.IgnoredProgrammingLanguage ||
			containerLanguage == common.NginxProgrammingLanguage {

			// TODO: this will make it look as if instrumentation device is applied,
			// which is incorrect
			modifiedContainers = append(modifiedContainers, container)
			continue
		}

		if container.Resources.Limits == nil {
			container.Resources.Limits = make(map[corev1.ResourceName]resource.Quantity)
		}

		modifiedContainers = append(modifiedContainers, container)
	}

	if modifiedContainers != nil {
		original.Spec.Containers = modifiedContainers
	}

	return deviceSkippedDueToOtherAgent, nil
}

func RevertInstrumentationDevices(original *corev1.PodTemplateSpec) bool {
	changed := false
	for _, container := range original.Spec.Containers {
		for resourceName := range container.Resources.Limits {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Limits, resourceName)
				changed = true
			}
		}
		// Is it needed?
		for resourceName := range container.Resources.Requests {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Requests, resourceName)
				changed = true
			}
		}
	}
	return changed
}

func getLanguageOfContainer(runtimeDetails []odigosv1.RuntimeDetailsByContainer, containerName string) common.ProgrammingLanguage {
	for _, rd := range runtimeDetails {
		if rd.ContainerName == containerName {
			return rd.Language
		}
	}

	return common.UnknownProgrammingLanguage
}

func getContainerOtherAgents(runtimeDetails []odigosv1.RuntimeDetailsByContainer, containerName string) *odigosv1.OtherAgent {
	for _, rd := range runtimeDetails {
		if rd.ContainerName == containerName {
			if rd.OtherAgent != nil && *rd.OtherAgent != (odigosv1.OtherAgent{}) {
				return rd.OtherAgent
			}
		}
	}
	return nil
}

func SetInjectInstrumentationLabel(original *corev1.PodTemplateSpec) {

	if original.Labels == nil {
		original.Labels = make(map[string]string)
	}
	original.Labels[k8sconsts.OdigosInjectInstrumentationLabel] = "true"
}

// RemoveInjectInstrumentationLabel removes the "odigos.io/inject-instrumentation" label if it exists.
func RemoveInjectInstrumentationLabel(original *corev1.PodTemplateSpec) bool {
	if original.Labels != nil {
		if _, ok := original.Labels[k8sconsts.OdigosInjectInstrumentationLabel]; ok {
			delete(original.Labels, k8sconsts.OdigosInjectInstrumentationLabel)
			return true
		}
	}
	return false
}
