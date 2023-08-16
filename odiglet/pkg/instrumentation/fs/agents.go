package fs

import (
	"os"
	"path/filepath"

	cp "github.com/otiai10/copy"
)

const (
	containerDir = "/instrumentations"
	hostDir      = "/var/odigos"
)

func CopyAgentsDirectoryToHost() error {

	// remove the current content of /var/odigos
	// as we want a fresh copy of instrumentation agents with no files leftover from previous odigos versions.
	// we cannot remove /var/odigos itself: "unlinkat /var/odigos: device or resource busy"
	// so we will just remove it's content
	entries, err := os.ReadDir(hostDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entryPath := filepath.Join(hostDir, entry.Name())
		err := os.RemoveAll(entryPath)
		if err != nil {
			return err
		}
	}

	err = cp.Copy(containerDir, hostDir, cp.Options{})
	if err != nil {
		return err
	}
	return nil
}
