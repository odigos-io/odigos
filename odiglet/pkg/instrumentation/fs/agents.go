package fs

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/odigos-io/odigos/odiglet/pkg/log"
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

	// We kept the following list of files to avoid removing instrumentations that are already loaded in the process memory
	filesToKeepMap := map[string]struct{}{
		"/var/odigos/nodejs-ebpf/build/Release/dtrace-injector-native.node":                            {},
		"/var/odigos/nodejs-ebpf/build/Release/obj.target/dtrace-injector-native.node":                 {},
		"/var/odigos/nodejs-ebpf/build/Release/.deps/Release/dtrace-injector-native.node.d":            {},
		"/var/odigos/nodejs-ebpf/build/Release/.deps/Release/obj.target/dtrace-injector-native.node.d": {},
		"/var/odigos/java-ebpf/tracing_probes.so":                                                      {},
		"/var/odigos/java-ext-ebpf/end_span_usdt.so":                                                   {},
		"/var/odigos/python-ebpf/pythonUSDT.abi3.so":                                                   {},
	}

	err := removeFilesInDir(hostDir, filesToKeepMap)
	if err != nil {
		log.Logger.Error(err, "Error removing instrumentation directory from host")
		return err
	}

	err = copyDirectories(containerDir, hostDir, filesToKeepMap)
	if err != nil {
		log.Logger.Error(err, "Error copying instrumentation directory to host")
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

func ShouldRecreateAllCFiles() bool {
	value, exists := os.LookupEnv("RECREATE_ALL_C_FILES")
	return exists && value == "true"
}
