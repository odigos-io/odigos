package envoverwrite

import (
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// original manifest values for the env vars of a workload
// This is specific to k8s as it assumes there is OriginalEnv per container
type OrigWorkloadEnvValues struct {
	origManifestValues   map[string]envOverwrite.OriginalEnv
	modifiedSinceCreated bool
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
		origManifestValues:   manifestValues,
		modifiedSinceCreated: false,
	}, nil
}

func (o *OrigWorkloadEnvValues) GetContainerStoredEnvs(containerName string) envOverwrite.OriginalEnv {
	return o.origManifestValues[containerName]
}

// this function is called when reverting a value back to it's original content.
// it removes the env, if exists, and returns the original value for the caller to populate back into the manifest.
func (o *OrigWorkloadEnvValues) RemoveOriginalValue(containerName string, envName string) (*string, bool) {
	if val, ok := o.origManifestValues[containerName][envName]; ok {
		delete(o.origManifestValues[containerName], envName)
		if len(o.origManifestValues[containerName]) == 0 {
			delete(o.origManifestValues, containerName)
		}
		o.modifiedSinceCreated = true
		return val, true
	}
	return nil, false
}

func (o *OrigWorkloadEnvValues) InsertOriginalValue(containerName string, envName string, val *string) {
	if _, ok := o.origManifestValues[containerName]; !ok {
		o.origManifestValues[containerName] = make(envOverwrite.OriginalEnv)
	}
	if _, alreadyExists := o.origManifestValues[containerName][envName]; alreadyExists {
		// we already have the original value for this env, will not update it
		// TODO: should we update it if the value is different?
		return
	}
	o.origManifestValues[containerName][envName] = val
	o.modifiedSinceCreated = true
}

// stores the original values back into the manifest annotations
// by modifying the annotations map of the input argument
func (o *OrigWorkloadEnvValues) SerializeToAnnotation(obj client.Object) error {
	if !o.modifiedSinceCreated {
		return nil
	}

	if len(o.origManifestValues) == 0 {
		// delete the annotation is there are no original values to store
		o.DeleteFromObj(obj)
		return nil
	}

	annotationContentBytes, err := json.Marshal(o.origManifestValues)
	if err != nil {
		// this should never happen, but if it does, we should log it and continue
		return fmt.Errorf("failed to marshal original env values: %v", err)
	}
	annotationEntryContent := string(annotationContentBytes)

	currentAnnotations := obj.GetAnnotations()
	if currentAnnotations == nil {
		currentAnnotations = make(map[string]string)
	}

	// write the original values back to the manifest annotations and update the object
	currentAnnotations[consts.ManifestEnvOriginalValAnnotation] = annotationEntryContent
	obj.SetAnnotations(currentAnnotations)
	return nil
}

func (o *OrigWorkloadEnvValues) DeleteFromObj(obj client.Object) {
	currentAnnotations := obj.GetAnnotations()
	if currentAnnotations == nil {
		return
	}

	delete(currentAnnotations, consts.ManifestEnvOriginalValAnnotation)
}
