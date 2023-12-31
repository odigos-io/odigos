package process

import (
	"fmt"
	"path"
	"strings"

	"github.com/fntlnz/mountinfo"
	procdiscovery "github.com/keyval-dev/odigos/procdiscovery/pkg/process"
)

func isPodContainerPredicate(podUID string, containerName string) func(string) bool {
	return func(procDirName string) bool {
		mi, err := mountinfo.GetMountInfo(path.Join("/proc", procDirName, "mountinfo"))
		if err != nil {
			return false
		}

		expectedMountRoot := fmt.Sprintf("%s/containers/%s", podUID, containerName)
		for _, m := range mi {
			root := m.Root
			if strings.Contains(root, expectedMountRoot) {
				return true
			}
		}

		return false
	}
}

func FindAllInContainer(podUID string, containerName string) ([]procdiscovery.Details, error) {
	return procdiscovery.FindAllProcesses(isPodContainerPredicate(podUID, containerName))
}
