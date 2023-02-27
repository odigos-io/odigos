package fs

import (
	cp "github.com/otiai10/copy"
)

const (
	containerDir = "/instrumentations"
	hostDir      = "/odigos"
)

func CopyAgentsDirectoryToHost() error {
	return cp.Copy(containerDir, hostDir)
}
