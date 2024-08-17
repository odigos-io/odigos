package workload

import "testing"

func TestGetRuntimeObjectName(t *testing.T) {
	name := "myworkload"
	kind := "Deployment"
	got := CalculateWorkloadRuntimeObjectName(name, kind)
	want := "deployment-myworkload"
	if got != want {
		t.Errorf("GetRuntimeObjectName() = %v, want %v", got, want)
	}
}
