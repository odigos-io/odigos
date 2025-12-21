package process

import (
	"fmt"
	"os"
	"strings"

	mount "github.com/moby/sys/mountinfo"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

func isInPodContainersBatchPredicate(podContainers []PodContainerUID) func(int) (PodContainerUID, bool) {
	expectedMountByPodContainer := make(map[PodContainerUID]string)
	for _, pc := range podContainers {
		expectedMount := fmt.Sprintf("%s/containers/%s/", pc.PodUID, pc.ContainerName)
		expectedMountByPodContainer[pc] = expectedMount
	}

	return func(pid int) (PodContainerUID, bool) {
		mountInfoFile := process.ProcFilePath(pid, "mountinfo")
		f, err := os.Open(mountInfoFile)
		if err != nil {
			return PodContainerUID{}, false
		}
		defer f.Close()

		relevantPodContainer := PodContainerUID{}
		infos, err := mount.GetMountsFromReader(f, func(m *mount.Info) (skip, stop bool) {
			for pc, mountPath := range expectedMountByPodContainer {
				if strings.Contains(m.Root, mountPath) {
					// Found the mount, add it and stop
					relevantPodContainer = pc
					return false, true
				}
			}

			// Keep looking
			return true, false
		})
		if err != nil {
			return PodContainerUID{}, false
		}
		if len(infos) > 0 {
			return relevantPodContainer, true
		}

		return PodContainerUID{}, false
	}
}

func isPodContainerPredicate(podUID string, containerName string) func(int) bool {

	// Added trailing slash to avoid substring collisions like "membership" matching "membership1".
	// Real m.Root ends with runtime ID (e.g., .../containers/membership/<runtime-id>), so exact match fails.
	// Using slash ensures we only match full "containers/<name>/" segments in mount paths.
	expectedMountRoot := fmt.Sprintf("%s/containers/%s/", podUID, containerName)

	return func(pid int) bool {
		mountInfoFile := process.ProcFilePath(pid, "mountinfo")
		f, err := os.Open(mountInfoFile)
		if err != nil {
			return false
		}
		defer f.Close()

		infos, err := mount.GetMountsFromReader(f, func(m *mount.Info) (skip, stop bool) {
			if strings.Contains(m.Root, expectedMountRoot) {
				// Found the mount, add it and stop
				return false, true
			}
			// Keep looking
			return true, false
		})
		if err != nil {
			return false
		}
		if len(infos) > 0 {
			return true
		}

		return false
	}
}

type PodContainerUID struct {
	PodUID, ContainerName string
}

func FindAllInContainer(podUID string, containerName string, runtimeDetectionEnvs map[string]struct{}) ([]process.Details, error) {
	pids, err :=  process.FindAllProcesses(isPodContainerPredicate(podUID, containerName))
	if err != nil {
		return nil, fmt.Errorf("failed to find processes for container %s :%w", containerName, err)
	}

	details := make([]process.Details, len(pids))
	for i, pid := range pids {
		details[i] = process.GetPidDetails(pid, runtimeDetectionEnvs)
	}

	return details, nil
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
