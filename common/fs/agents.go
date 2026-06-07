package fs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

const (
	chrootDir      = "/host"
	semanagePath   = "/sbin/semanage"
	restoreconPath = "/sbin/restorecon"
)

func CopyAgentsDirectoryToHost(srcDir, dstDir string) error {
	logger := commonlogger.LoggerCompat().With("subsystem", "agents")
	startTime := time.Now()
	empty, err := isDirEmptyOrNotExist(dstDir)
	if err != nil {
		return fmt.Errorf("failed to inspect destination: %w", err)
	}

	if empty {
		// if empty, we can just copy the directory to the host
		logger.Info("Odigos agents directory is empty, copying agents directory to host")
		err = CopyDirectories(srcDir, dstDir, nil)
		if err != nil {
			logger.Error("Error copying instrumentation directory to host", "err", err)
			return err
		}
	} else {
		logger.Info("Odigos agents directory is not empty, performing safe upgrade")
		excludes, err := ProcessCriticalFiles(criticalFiles, srcDir, dstDir)
		if err != nil {
			logger.Error("Error processing critical files", "err", err)
		}

		if err := CopyDirectories(srcDir, dstDir, excludes); err != nil {
			logger.Error("Error syncing agents directory to host", "err", err)
			return err
		}
	}

	// temporary workaround for dotnet.
	// dotnet used to have directories containing the arch suffix (linux-glibc-arm64).
	// this works will with virtual device that knows the arch it is running on.
	// however, the webhook cannot know in advance which arch the pod is going to run on.
	// thus, the directory names are renamed so they do not contain the arch suffix (linux-glibc)
	// which can be used by the webhook.
	// The following link is a temporary support for the deprecated dotnet virtual devices.
	// TODO: remove this once we delete the virtual devices.
	err = createDotnetDeprecatedDirectories(path.Join(dstDir, "dotnet"))
	if err != nil {
		logger.Error("Error creating dotnet deprecated directories", "err", err)
		return err
	}

	logger.Info("Odigos agents directory copied to host", "elapsed", time.Since(startTime))

	return nil
}

// ApplyOpenShiftSELinuxSettings makes auto-instrumentation agents readable by containers on RHEL hosts.
// Note: This function calls chroot to use the host's PATH to execute selinux commands. Calling it will
// affect the odiglet running process's apparent filesystem.
func ApplyOpenShiftSELinuxSettings(dstDir string) error {
	// Check if the semanage command exists when running on RHEL/CoreOS
	logger := commonlogger.LoggerCompat().With("subsystem", "agents")
	logger.Info("Applying selinux settings to host")
	_, err := exec.LookPath(filepath.Join(chrootDir, semanagePath))
	if err == nil {
		err = syscall.Chroot(chrootDir)
		if err != nil {
			logger.Error("Error chrooting to host", "err", err)
		}

		// list existing semanage rules to check if Odigos has already been set
		cmd := exec.CommandContext(context.Background(), semanagePath, "fcontext", "-l")
		var out bytes.Buffer
		cmd.Stdout = &out

		err := cmd.Run()
		if err != nil {
			logger.Error("Error executing semanage", "err", err)
			return err
		}

		pattern := regexp.MustCompile(`/var/odigos(\(/.\*\)\?)?\s+.*container_ro_file_t`)
		if pattern.Match(out.Bytes()) {
			logger.Info("Rule for /var/odigos already exists with container_ro_file_t.")
			return nil
		}

		// Run the semanage command to add the new directory to the container_ro_file_t context
		// semanage writes SELinux config to host
		cmd = exec.CommandContext(context.Background(), semanagePath, "fcontext", "-a", "-t", "container_ro_file_t", "/var/odigos(/.*)?")
		stdoutBytes, err := cmd.CombinedOutput()
		if err != nil {
			logger.Error("Error running semanage command", "err", err, "stdout", string(stdoutBytes))
			if strings.Contains(string(stdoutBytes), "already defined") {
				// some versions of selinux return an error when trying to set fcontext where it already exists
				// if that's the case, we don't need to return an error here
				return nil
			}
			return err
		}

		// Check if the restorecon command exists when running on RHEL/CoreOS
		// restorecon applies the SELinux settings we just created to the host
		// And we are already chrooted to the host path, so we can just look for restoreconPath now
		_, err = exec.LookPath(restoreconPath)
		if err == nil {
			// Run the restorecon command to apply the new context
			cmd := exec.CommandContext(context.Background(), restoreconPath, "-r", dstDir)
			err = cmd.Run()
			if err != nil {
				logger.Error("Error running restorecon command", "err", err)
				return err
			}
		} else {
			logger.Error("Unable to find restorecon path", "err", err)
			return err
		}
	} else {
		logger.Info("Unable to find semanage path, possibly not on RHEL host")
	}
	return nil
}

func isDirEmptyOrNotExist(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("not a directory: %s", dir)
	}
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			commonlogger.LoggerCompat().Error("Error closing file", "err", err)
		}
	}()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
