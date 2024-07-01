//go:build !linux

package process

import procdiscovery "github.com/odigos-io/odigos/procdiscovery/pkg/process"

// These functions are stubs for non-linux platforms to allow running tests on them.

func isPodContainerPredicate(_ string, _ string) func(string) bool {
	return func(procDirName string) bool {
		return false
	}
}

func FindAllInContainer(podUID string, containerName string) ([]procdiscovery.Details, error) {
	return nil, nil
}