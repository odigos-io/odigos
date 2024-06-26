package process

import (
	"fmt"
	"os"
	"path"
	"strings"

	mount "github.com/moby/sys/mountinfo"
	procdiscovery "github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

func isPodContainerPredicate(podUID string, containerName string) func(string) bool {
	expectedMountRoot := fmt.Sprintf("%s/containers/%s", podUID, containerName)

	return func(procDirName string) bool {
		mountInfoFile := path.Join("/proc", procDirName, "mountinfo")
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

func FindAllInContainer(podUID string, containerName string) ([]procdiscovery.Details, error) {
	return procdiscovery.FindAllProcesses(isPodContainerPredicate(podUID, containerName))
}
