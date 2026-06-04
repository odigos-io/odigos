package graph

import (
	"cmp"
	"slices"
	"strconv"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// collectEffectiveDetectedLanguages returns the sorted, unique list of programming
// languages across all containers of an InstrumentationConfig, preferring the
// user-defined override (Spec.ContainersOverrides[].RuntimeInfo) over the
// auto-detected runtime info (Status.RuntimeDetailsByContainer). Ignored
// containers and unknown languages are excluded, matching the client-side
// `getEffectiveLanguage` resolution.
func collectEffectiveDetectedLanguages(ic *odigosv1.InstrumentationConfig, ignoredContainers map[string]struct{}) []model.ProgrammingLanguage {
	if ic == nil {
		return nil
	}

	// ic.RuntimeDetailsByContainer() returns one entry per container in the workload,
	// preferring the override's RuntimeInfo over the automatic detection result.
	effective := ic.RuntimeDetailsByContainer()

	uniqueLanguages := make(map[common.ProgrammingLanguage]struct{})
	for containerName, details := range effective {
		if details == nil {
			continue
		}
		if _, ignored := ignoredContainers[containerName]; ignored {
			continue
		}
		if details.Language == common.UnknownProgrammingLanguage {
			continue
		}
		uniqueLanguages[details.Language] = struct{}{}
	}

	// Defensive fallback for containers that have a detected runtime but are
	// missing from ContainersOverrides — ContainersOverrides should list every
	// container, but we don't want to silently drop detected languages if it doesn't.
	for _, container := range ic.Status.RuntimeDetailsByContainer {
		if _, ok := effective[container.ContainerName]; ok {
			continue
		}
		if _, ignored := ignoredContainers[container.ContainerName]; ignored {
			continue
		}
		if container.Language == common.UnknownProgrammingLanguage {
			continue
		}
		uniqueLanguages[container.Language] = struct{}{}
	}

	detectedLanguages := make([]model.ProgrammingLanguage, 0, len(uniqueLanguages))
	for language := range uniqueLanguages {
		detectedLanguages = append(detectedLanguages, model.ProgrammingLanguage(language))
	}
	slices.Sort(detectedLanguages)
	return detectedLanguages
}

// componentToInstrumentation maps an InstrumentationLibraryStatus from an
// instrumentation instance CR into the GraphQL `K8sWorkloadPodContainerProcessInstrumentation`
// shape. Centralized so the workload-container summary and the per-process
// resolver expose the same fields (type, healthy, message, lastStatusTime,
// nonIdentifyingAttributes) the legacy `instrumentationInstanceComponents`
// query used to surface.
func componentToInstrumentation(component odigosv1.InstrumentationLibraryStatus) *model.K8sWorkloadPodContainerProcessInstrumentation {
	var typeStr *string
	if component.Type != "" {
		t := string(component.Type)
		typeStr = &t
	}

	var lastStatusTime *string
	if !component.LastStatusTime.IsZero() {
		t := component.LastStatusTime.Format(time.RFC3339)
		lastStatusTime = &t
	}

	var message *string
	if component.Message != "" {
		m := component.Message
		message = &m
	}

	var isStandardLibrary *bool
	nonIdentifyingAttributes := make([]*model.NonIdentifyingAttribute, 0, len(component.NonIdentifyingAttributes))
	for _, attribute := range component.NonIdentifyingAttributes {
		nonIdentifyingAttributes = append(nonIdentifyingAttributes, &model.NonIdentifyingAttribute{
			Key:   attribute.Key,
			Value: attribute.Value,
		})
		if attribute.Key == "is_standard_lib" {
			valBool := attribute.Value == "true"
			isStandardLibrary = &valBool
		}
	}

	return &model.K8sWorkloadPodContainerProcessInstrumentation{
		Name:                     component.Name,
		Type:                     typeStr,
		Healthy:                  component.Healthy,
		Message:                  message,
		LastStatusTime:           lastStatusTime,
		IsStandardLibrary:        isStandardLibrary,
		NonIdentifyingAttributes: nonIdentifyingAttributes,
	}
}

// compareProcessesByPid orders two processes by their reported PID — preferring
// process.pid, falling back to process.vpid — using numeric ordering when both
// values are valid integers (so "10" sorts after "2"), and lexicographic
// otherwise. Mirrors the fallback rules the UI used to apply on the client.
func compareProcessesByPid(a, b *model.K8sWorkloadPodContainerProcess) int {
	aPid := processPidFromAttributes(a.IdentifyingAttributes)
	bPid := processPidFromAttributes(b.IdentifyingAttributes)

	aInt, aErr := strconv.Atoi(aPid)
	bInt, bErr := strconv.Atoi(bPid)
	if aErr == nil && bErr == nil {
		return cmp.Compare(aInt, bInt)
	}
	return cmp.Compare(aPid, bPid)
}

// processPidFromAttributes returns the PID-like value used to order processes
// deterministically across re-fetches. Prefers process.pid, falling back to
// process.vpid (the virtual PID emitted by some agents) so processes that only
// report vpid still sort meaningfully instead of clustering under the empty
// string.
func processPidFromAttributes(attrs []*model.K8sWorkloadPodContainerProcessAttribute) string {
	var vpid string
	for _, attr := range attrs {
		if attr == nil {
			continue
		}
		switch attr.Name {
		case processAttributeNamePid:
			return attr.Value
		case processAttributeNameVpid:
			vpid = attr.Value
		}
	}
	return vpid
}
