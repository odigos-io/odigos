package graph

import (
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func mergeContainerOverride(existing []v1alpha1.ContainerOverride, containerName string, patch model.PatchSourceRequestInput) ([]v1alpha1.ContainerOverride, error) {
	containerOverrides := make([]v1alpha1.ContainerOverride, 0, len(existing)+1)
	containerOverride := v1alpha1.ContainerOverride{ContainerName: containerName}

	for _, override := range existing {
		if override.ContainerName == containerName {
			containerOverride = override
			continue
		}
		containerOverrides = append(containerOverrides, override)
	}

	if patch.Language != nil {
		if *patch.Language == "" {
			containerOverride.RuntimeInfo = nil
		} else {
			runtimeVersion := ""
			if patch.Version != nil {
				runtimeVersion = *patch.Version
			} else if containerOverride.RuntimeInfo != nil {
				runtimeVersion = containerOverride.RuntimeInfo.RuntimeVersion
			}
			if runtimeVersion != "" && common.GetVersion(runtimeVersion) == nil {
				return nil, fmt.Errorf("invalid runtime version: %s", runtimeVersion)
			}
			containerOverride.RuntimeInfo = &v1alpha1.RuntimeDetailsByContainer{
				ContainerName:  containerName,
				Language:       common.ProgrammingLanguage(*patch.Language),
				RuntimeVersion: runtimeVersion,
			}
		}
	} else if patch.Version != nil && containerOverride.RuntimeInfo != nil {
		if *patch.Version != "" && common.GetVersion(*patch.Version) == nil {
			return nil, fmt.Errorf("invalid runtime version: %s", *patch.Version)
		}
		runtimeInfo := *containerOverride.RuntimeInfo
		runtimeInfo.RuntimeVersion = *patch.Version
		containerOverride.RuntimeInfo = &runtimeInfo
	}

	if patch.OtelDistroName != nil {
		containerOverride.OtelDistroName = patch.OtelDistroName
	}
	if patch.AllowConcurrentAgents != nil {
		containerOverride.AllowConcurrentAgents = patch.AllowConcurrentAgents
	}

	return append(containerOverrides, containerOverride), nil
}
