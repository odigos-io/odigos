package workload

import "testing"

func TestGetRuntimeObjectName(t *testing.T) {
	name := "myworkload"
	kind := "Deployment"
	got := GetRuntimeObjectName(name, kind)
	want := "deployment-myworkload"
	if got != want {
		t.Errorf("GetRuntimeObjectName() = %v, want %v", got, want)
	}
}

func TestGetTargetFromRuntimeName(t *testing.T) {
	name := "myworkload"
	kind := "Deployment"
	runtimeName := GetRuntimeObjectName(name, kind)
	gotName, gotKind, err := GetWorkloadInfoRuntimeName(runtimeName)
	if err != nil {
		t.Errorf("GetTargetFromRuntimeName() error = %v", err)
	}
	if gotName != name {
		t.Errorf("GetTargetFromRuntimeName() gotName = %v, want %v", gotName, name)
	}
	if gotKind != kind {
		t.Errorf("GetTargetFromRuntimeName() gotKind = %v, want %v", gotKind, kind)
	}
}

func TestGetTargetFromRuntimeName_HyphenInName(t *testing.T) {
	name := "my-workload"
	kind := "Deployment"
	runtimeName := GetRuntimeObjectName(name, kind)
	gotName, gotKind, err := GetWorkloadInfoRuntimeName(runtimeName)
	if err != nil {
		t.Errorf("GetTargetFromRuntimeName() error = %v", err)
	}
	if gotName != name {
		t.Errorf("GetTargetFromRuntimeName() gotName = %v, want %v", gotName, name)
	}
	if gotKind != kind {
		t.Errorf("GetTargetFromRuntimeName() gotKind = %v, want %v", gotKind, kind)
	}
}

func TestGetTargetFromRuntimeName_EmptyName(t *testing.T) {
	_, _, err := GetWorkloadInfoRuntimeName("")
	if err == nil {
		t.Errorf("GetTargetFromRuntimeName() error = %v, want %v", err, "invalid runtime name")
	}
}

func TestGetTargetFromRuntimeName_InvalidKind(t *testing.T) {
	_, _, err := GetWorkloadInfoRuntimeName("invalidworkloadkind-myworkload")
	if err == nil {
		t.Errorf("GetTargetFromRuntimeName() error = %v, want %v", err, "unknown kind")
	}
}

func TestGetTargetFromRuntimeName_AllSupportedKinds(t *testing.T) {
	testCases := []struct {
		runtimeName  string
		expectedKind string
	}{
		{"deployment-myworkload", "Deployment"},
		{"statefulset-myworkload", "StatefulSet"},
		{"daemonset-myworkload", "DaemonSet"},
	}

	for _, tc := range testCases {
		_, gotKind, err := GetWorkloadInfoRuntimeName(tc.runtimeName)
		if err != nil {
			t.Errorf("GetTargetFromRuntimeName() with input %v error = %v", tc.runtimeName, err)
		}
		if gotKind != tc.expectedKind {
			t.Errorf("GetTargetFromRuntimeName() with input %v gotKind = %v, want %v", tc.runtimeName, gotKind, tc.expectedKind)
		}
	}
}
