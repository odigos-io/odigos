package graph

import (
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func TestMergeContainerOverridePreservesOmittedFields(t *testing.T) {
	distroName := "java-community"
	allowConcurrentAgents := true
	existing := []v1alpha1.ContainerOverride{
		{
			ContainerName: "sidecar",
			OtelDistroName: func() *string {
				value := "python-community"
				return &value
			}(),
		},
		{
			ContainerName: "app",
			RuntimeInfo: &v1alpha1.RuntimeDetailsByContainer{
				ContainerName:  "app",
				Language:       common.JavaProgrammingLanguage,
				RuntimeVersion: "17",
			},
			OtelDistroName:        &distroName,
			AllowConcurrentAgents: &allowConcurrentAgents,
		},
	}
	disallowConcurrentAgents := false

	got, err := mergeContainerOverride(existing, "app", model.PatchSourceRequestInput{
		AllowConcurrentAgents: &disallowConcurrentAgents,
	})
	if err != nil {
		t.Fatalf("mergeContainerOverride returned an error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected two overrides, got %d", len(got))
	}
	if !reflect.DeepEqual(got[0], existing[0]) {
		t.Fatalf("unrelated container override changed: got %#v, want %#v", got[0], existing[0])
	}

	appOverride := got[1]
	if !reflect.DeepEqual(appOverride.RuntimeInfo, existing[1].RuntimeInfo) {
		t.Fatalf("runtime override changed: got %#v, want %#v", appOverride.RuntimeInfo, existing[1].RuntimeInfo)
	}
	if !reflect.DeepEqual(appOverride.OtelDistroName, existing[1].OtelDistroName) {
		t.Fatalf("distro override changed: got %#v, want %#v", appOverride.OtelDistroName, existing[1].OtelDistroName)
	}
	if appOverride.AllowConcurrentAgents == nil || *appOverride.AllowConcurrentAgents {
		t.Fatalf("expected allowConcurrentAgents to be explicitly false, got %#v", appOverride.AllowConcurrentAgents)
	}
}

func TestMergeContainerOverridePreservesConcurrentAgentSetting(t *testing.T) {
	allowConcurrentAgents := true
	language := string(common.PythonProgrammingLanguage)
	version := "3.12"
	distroName := "python-community"

	got, err := mergeContainerOverride([]v1alpha1.ContainerOverride{
		{
			ContainerName:         "app",
			AllowConcurrentAgents: &allowConcurrentAgents,
		},
	}, "app", model.PatchSourceRequestInput{
		Language:       &language,
		Version:        &version,
		OtelDistroName: &distroName,
	})
	if err != nil {
		t.Fatalf("mergeContainerOverride returned an error: %v", err)
	}

	appOverride := got[0]
	if appOverride.AllowConcurrentAgents == nil || !*appOverride.AllowConcurrentAgents {
		t.Fatalf("expected allowConcurrentAgents to remain true, got %#v", appOverride.AllowConcurrentAgents)
	}
	if appOverride.RuntimeInfo == nil || appOverride.RuntimeInfo.Language != common.PythonProgrammingLanguage || appOverride.RuntimeInfo.RuntimeVersion != version {
		t.Fatalf("unexpected runtime override: %#v", appOverride.RuntimeInfo)
	}
	if appOverride.OtelDistroName == nil || *appOverride.OtelDistroName != distroName {
		t.Fatalf("unexpected distro override: %#v", appOverride.OtelDistroName)
	}
}
