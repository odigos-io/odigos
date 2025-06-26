package envoverwrite

import (
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"
)

// Deprecated: Used for migration purposes only.
// remove in odigos v1.1
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
