package runtimes

import (
	"errors"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/moby/sys/mountinfo"
	"strings"
)

var ErrNoMountFound = errors.New("no mount found")

type containerdRuntime struct{}

func (c *containerdRuntime) Name() ContainerRuntime {
	return Containerd
}

func (c *containerdRuntime) GetFileSystemPath(containerId string) (string, error) {
	// Containerd overlayfs mount path include the container id, so we can simply iterate over the existing mounts
	// and find the one that matches the container id
	mounts, err := mountinfo.GetMounts(func(info *mountinfo.Info) (skip, stop bool) {
		if strings.Contains(info.Mountpoint, containerId) {
			return false, true
		}

		return true, false
	})

	if err != nil {
		log.Logger.Error(err, "Failed to get mounts")
		return "", err
	}

	if len(mounts) == 0 {
		err = ErrNoMountFound
		log.Logger.Error(err, "No mounts found")
		return "", err
	}

	return mounts[0].Mountpoint, nil
}
