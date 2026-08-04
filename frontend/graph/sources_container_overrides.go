package graph

import "github.com/odigos-io/odigos/api/odigos/v1alpha1"

// buildUpdatedContainerOverrides replaces the override for containerName while
// preserving fields that PatchSourceRequestInput cannot express (today:
// AllowConcurrentAgents). Sibling containers are left unchanged.
func buildUpdatedContainerOverrides(
	existing []v1alpha1.ContainerOverride,
	containerName string,
	otelDistroName *string,
	runtimeInfo *v1alpha1.RuntimeDetailsByContainer,
) []v1alpha1.ContainerOverride {
	updated := make([]v1alpha1.ContainerOverride, 0, len(existing)+1)
	var preservedAllowConcurrentAgents *bool
	for _, override := range existing {
		if override.ContainerName != containerName {
			updated = append(updated, override)
			continue
		}
		preservedAllowConcurrentAgents = override.AllowConcurrentAgents
	}
	updated = append(updated, v1alpha1.ContainerOverride{
		ContainerName:         containerName,
		OtelDistroName:        otelDistroName,
		RuntimeInfo:           runtimeInfo,
		AllowConcurrentAgents: preservedAllowConcurrentAgents,
	})
	return updated
}
