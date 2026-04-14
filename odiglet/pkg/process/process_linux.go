package process

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"

	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PodContainerUID struct {
	PodUID, ContainerName string
}

// GroupByPodContainer groups all current active processes by (podUID, containerName)
// using the provided list of PodContainerUIDs to filter relevant processes.
// Processes that do not belong to any of the provided PodContainerUIDs are ignored.
//
// Uses a single pass over /proc: for each PID, reads mountinfo once and matches
// against all expected containers. This avoids the allocation churn of reading
// mountinfo inside a per-PID predicate closure.
func GroupByPodContainer(pcs []PodContainerUID) (map[PodContainerUID]map[int]struct{}, error) {
	expectedMounts := make(map[PodContainerUID][]byte, len(pcs))
	for _, pc := range pcs {
		expectedMounts[pc] = []byte(fmt.Sprintf("%s/containers/%s/", pc.PodUID, pc.ContainerName))
	}

	dirs, err := os.ReadDir(process.HostProcDir())
	if err != nil {
		return nil, fmt.Errorf("failed to read proc dir: %w", err)
	}

	result := make(map[PodContainerUID]map[int]struct{})
	for _, di := range dirs {
		if !di.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(di.Name())
		if err != nil {
			continue
		}

		f, err := os.Open(process.ProcFilePath(pid, "mountinfo"))
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		found := false
		for scanner.Scan() {
			line := scanner.Bytes()
			for pc, mount := range expectedMounts {
				if bytes.Contains(line, mount) {
					if result[pc] == nil {
						result[pc] = make(map[int]struct{})
					}
					result[pc][pid] = struct{}{}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		f.Close()
	}

	return result, nil
}
