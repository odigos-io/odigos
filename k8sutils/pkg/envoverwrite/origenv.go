package envoverwrite

import (
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
)

// Deprecated. Used for migration purposes only.
// remove in odigos v1.1
type OrigWorkloadEnvValues struct {
	OrigManifestValues map[string]envOverwrite.OriginalEnv
}

func NewOrigWorkloadEnvValues(workloadAnnotations map[string]string) (*OrigWorkloadEnvValues, error) {
	manifestValues := make(map[string]envOverwrite.OriginalEnv)
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

func (o *OrigWorkloadEnvValues) GetContainerStoredEnvs(containerName string) envOverwrite.OriginalEnv {
	return o.OrigManifestValues[containerName]
}

// this function is called when reverting a value back to it's original content.
// it removes the env, if exists, and returns the original value for the caller to populate back into the manifest.
func (o *OrigWorkloadEnvValues) RemoveOriginalValue(containerName, envName string) (*string, bool) {
	if val, ok := o.OrigManifestValues[containerName][envName]; ok {
		delete(o.OrigManifestValues[containerName], envName)
		if len(o.OrigManifestValues[containerName]) == 0 {
			delete(o.OrigManifestValues, containerName)
		}
		return val, true
	}
	return nil, false
}

func (o *OrigWorkloadEnvValues) InsertOriginalValue(containerName, envName string, val *string) {
	if _, ok := o.OrigManifestValues[containerName]; !ok {
		o.OrigManifestValues[containerName] = make(envOverwrite.OriginalEnv)
	}
	if _, alreadyExists := o.OrigManifestValues[containerName][envName]; alreadyExists {
		// we already have the original value for this env, will not update it
		// TODO: should we update it if the value is different?
		return
	}
	o.OrigManifestValues[containerName][envName] = val
}
