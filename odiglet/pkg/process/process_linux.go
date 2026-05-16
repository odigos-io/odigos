package process

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

func isInPodContainersBatchPredicate(podContainers []PodContainerUID) func(int) (PodContainerUID, bool) {
	expectedMountByPodContainer := make(map[PodContainerUID][]byte)
	for _, pc := range podContainers {
		expectedMount := fmt.Sprintf("%s/containers/%s/", pc.PodUID, pc.ContainerName)
		expectedMountByPodContainer[pc] = []byte(expectedMount)
	}

	return func(pid int) (PodContainerUID, bool) {
		mountInfoFile := process.ProcFilePath(pid, "mountinfo")
		f, err := os.Open(mountInfoFile)
		if err != nil {
			return PodContainerUID{}, false
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			for pc, mountPath := range expectedMountByPodContainer {
				if bytes.Contains(scanner.Bytes(), mountPath) {
					return pc, true
				}
			}
		}
		return PodContainerUID{}, false
	}
}

type PodContainerUID struct {
	PodUID, ContainerName string
}

// GroupByPodContainer groups all the current active processes by (podUID, containerName) using the provided list of PodContainerUIDs to filter relevant processes.
// Processes that do not belong to any of the provided PodContainerUIDs are ignored.
func GroupByPodContainer(pcs []PodContainerUID) (map[PodContainerUID]map[int]struct{}, error) {
	groups, err := process.Group(isInPodContainersBatchPredicate(pcs))
	if err != nil {
		return nil, fmt.Errorf("failed to group processes by (pod, container) :%w", err)
	}

	return groups, nil
}
