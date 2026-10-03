package predicates

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func instrumentationConfigWithRuntimeDetails(details ...odigosv1.RuntimeDetailsByContainer) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: details,
		},
	}
}

func TestRuntimeDetailsChangedPredicate_Update_ReorderedContainersNoChange(t *testing.T) {
	old := instrumentationConfigWithRuntimeDetails(
		odigosv1.RuntimeDetailsByContainer{ContainerName: "a", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.20"},
		odigosv1.RuntimeDetailsByContainer{ContainerName: "b", Language: common.JavaProgrammingLanguage, RuntimeVersion: "17"},
	)
	// same containers, same details, just reordered - should NOT be considered a change.
	newObj := instrumentationConfigWithRuntimeDetails(
		odigosv1.RuntimeDetailsByContainer{ContainerName: "b", Language: common.JavaProgrammingLanguage, RuntimeVersion: "17"},
		odigosv1.RuntimeDetailsByContainer{ContainerName: "a", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.20"},
	)

	p := RuntimeDetailsChangedPredicate{}
	if got := p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: newObj}); got != false {
		t.Errorf("expected no change to be detected for reordered but otherwise identical containers, got %v", got)
	}
}

func TestRuntimeDetailsChangedPredicate_Update_SwappedContainersDetectsRealChange(t *testing.T) {
	old := instrumentationConfigWithRuntimeDetails(
		odigosv1.RuntimeDetailsByContainer{ContainerName: "a", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.20"},
		odigosv1.RuntimeDetailsByContainer{ContainerName: "b", Language: common.JavaProgrammingLanguage, RuntimeVersion: "17"},
	)
	// container "a" now has a different runtime version, but position-based comparison
	// (old[0] vs new[0]) would compare it against "b" and miss the real change.
	newObj := instrumentationConfigWithRuntimeDetails(
		odigosv1.RuntimeDetailsByContainer{ContainerName: "b", Language: common.JavaProgrammingLanguage, RuntimeVersion: "17"},
		odigosv1.RuntimeDetailsByContainer{ContainerName: "a", Language: common.GoProgrammingLanguage, RuntimeVersion: "1.21"},
	)

	p := RuntimeDetailsChangedPredicate{}
	if got := p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: newObj}); got != true {
		t.Errorf("expected change to be detected for container 'a' runtime version bump, got %v", got)
	}
}

func TestRuntimeDetailsChangedPredicate_Update_ContainerSetChanged(t *testing.T) {
	old := instrumentationConfigWithRuntimeDetails(
		odigosv1.RuntimeDetailsByContainer{ContainerName: "a", Language: common.GoProgrammingLanguage},
	)
	newObj := instrumentationConfigWithRuntimeDetails(
		odigosv1.RuntimeDetailsByContainer{ContainerName: "b", Language: common.GoProgrammingLanguage},
	)

	p := RuntimeDetailsChangedPredicate{}
	if got := p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: newObj}); got != true {
		t.Errorf("expected change to be detected when the container name changes, got %v", got)
	}
}
