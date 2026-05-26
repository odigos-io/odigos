package process

import (
	"os"
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestGroupByPodContainerFallsBackForMissingCgroupResults(t *testing.T) {
	prevLayout := hostCgroupLayout
	prevFallback := groupByProcMountInfoFunc
	t.Cleanup(func() {
		hostCgroupLayout = prevLayout
		groupByProcMountInfoFunc = prevFallback
	})

	cgroupRoot := t.TempDir()
	hostCgroupLayout = cgroupLayout{
		Valid: true,
		root:  cgroupRoot,
	}

	resolvedPC := PodContainer{
		PodContainerKey: PodContainerKey{PodUID: "resolved", ContainerName: "app"},
		QOSClass:        corev1.PodQOSGuaranteed,
		ContainerID:     "containerd://resolved-container",
	}
	missingPC := PodContainer{
		PodContainerKey: PodContainerKey{PodUID: "missing", ContainerName: "app"},
		QOSClass:        corev1.PodQOSGuaranteed,
		ContainerID:     "containerd://missing-container",
	}

	resolvedCgroupDir := filepath.Join(cgroupRoot, "kubepods", "podresolved", "resolved-container")
	if err := os.MkdirAll(resolvedCgroupDir, 0o755); err != nil {
		t.Fatalf("mkdir cgroup dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resolvedCgroupDir, "cgroup.procs"), []byte("101\n"), 0o644); err != nil {
		t.Fatalf("write cgroup.procs: %v", err)
	}

	groupByProcMountInfoFunc = func(pcs []PodContainer) (map[PodContainerKey]map[int]struct{}, error) {
		if len(pcs) != 1 || pcs[0].PodContainerKey != missingPC.PodContainerKey {
			t.Fatalf("fallback called with %#v, want only missing pod-container", pcs)
		}
		return map[PodContainerKey]map[int]struct{}{
			missingPC.PodContainerKey: {202: {}},
		}, nil
	}

	groups, err := GroupByPodContainer([]PodContainer{resolvedPC, missingPC})
	if err != nil {
		t.Fatalf("GroupByPodContainer: %v", err)
	}
	if _, ok := groups[resolvedPC.PodContainerKey][101]; !ok {
		t.Fatalf("expected cgroup PID for resolved pod-container, got %#v", groups)
	}
	if _, ok := groups[missingPC.PodContainerKey][202]; !ok {
		t.Fatalf("expected proc fallback PID for missing pod-container, got %#v", groups)
	}
}
