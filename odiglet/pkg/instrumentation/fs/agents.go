package fs

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/odigos-io/odigos/odiglet/pkg/log"
	cp "github.com/otiai10/copy"
)

const (
	containerDir   = "/instrumentations"
	hostDir        = "/var/odigos"
	chrootDir      = "/host"
	semanagePath   = "/sbin/semanage"
	restoreconPath = "/sbin/restorecon"
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

	// Check if the semanage command exists when running on RHEL/CoreOS
	_, err = exec.LookPath(filepath.Join(chrootDir, semanagePath))
	if err == nil {
		// Run the semanage command to add the new directory to the container_ro_file_t context
		cmd := exec.Command(semanagePath, "fcontext", "-a", "-t", "container_ro_file_t", "/var/odigos(/.*)?")
		syscall.Chroot(chrootDir)
		err = cmd.Run()
		if err != nil {
			log.Logger.Error(err, "Error running semanage command")
		}
	}

	// Check if the restorecon command exists when running on RHEL/CoreOS
	_, err = exec.LookPath(filepath.Join(chrootDir, restoreconPath))
	if err == nil {
		// Run the restorecon command to apply the new context
		cmd := exec.Command(restoreconPath, "-r", "/var/odigos")
		syscall.Chroot(chrootDir)
		err = cmd.Run()
		if err != nil {
			log.Logger.Error(err, "Error running restorecon command")
		}
	}

	return nil
}
