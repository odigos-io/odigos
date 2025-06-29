package deprecated_envoverwrite

import (
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"
)

// Odigos will not modify any environment for workload objects (deployments, daemonsets, etc.)
// since Jan 2025, and should revert any changes made to the environment variables for any version upgrade after that.
// The deprecated envoverwrite mechanism, however, is still left here for some extended time,
// just to make sure if someone uses an old version of odigos, we will cleanup after ourselves on uninstall.
//
// This is a temporary solution and will be removed one day.
type OrigWorkloadEnvValues struct {
	OrigManifestValues map[string]map[string]*string
}

func NewOrigWorkloadEnvValues(workloadAnnotations map[string]string) (*OrigWorkloadEnvValues, error) {
	manifestValues := make(map[string]map[string]*string)
	if workloadAnnotations != nil {
		if currentEnvAnnotation, ok := workloadAnnotations[consts.ManifestEnvOriginalValAnnotation]; ok {
			err := json.Unmarshal([]byte(currentEnvAnnotation), &manifestValues)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal manifest env original annotation: %v", err)
			}
		}
	}

	return &OrigWorkloadEnvValues{
		OrigManifestValues: manifestValues,
	}, nil
}

func (o *OrigWorkloadEnvValues) GetContainerStoredEnvs(containerName string) map[string]*string {
	return o.OrigManifestValues[containerName]
}
