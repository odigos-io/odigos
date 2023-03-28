package fs

import (
	cp "github.com/otiai10/copy"
)

const (
	containerDir = "/instrumentations"
	hostDir      = "/var/odigos"
)

func CopyAgentsDirectoryToHost() error {
	return cp.Copy(containerDir, hostDir)
}
