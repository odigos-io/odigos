package csi

import (
	"fmt"
	"os/exec"
)

const (
	// nsenterPath is the path to the nsenter binary
	nsenterPath = "/usr/bin/nsenter"
)

// runInHostMountNS executes a command in the host's mount namespace using nsenter.
// This is more reliable than using setns() directly, as nsenter handles
// the namespace switching in a separate process.
func runInHostMountNS(name string, args ...string) ([]byte, error) {
	// Build nsenter command to run in host's mount namespace
	// -m: enter mount namespace
	// -t 1: target PID 1 (host's init process)
	nsenterArgs := []string{"-m", "-t", "1", name}
	nsenterArgs = append(nsenterArgs, args...)

	cmd := exec.Command(nsenterPath, nsenterArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("nsenter command failed: %w, output: %s", err, string(output))
	}
	return output, nil
}

// mountBindInHostNS performs a read-only bind mount in the host's mount namespace.
// This makes the mount visible to kubelet without requiring Bidirectional
// mount propagation (and thus without requiring privileged mode).
// Uses nsenter to execute mount commands in the host namespace.
func mountBindInHostNS(source, target string) error {
	// Create target directory using nsenter (in host namespace)
	if _, err := runInHostMountNS("mkdir", "-p", target); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", target, err)
	}

	// Perform the bind mount using nsenter
	if _, err := runInHostMountNS("mount", "--bind", source, target); err != nil {
		return fmt.Errorf("failed to bind mount %s to %s: %w", source, target, err)
	}

	// Make the mount read-only
	if _, err := runInHostMountNS("mount", "-o", "remount,ro,bind", target); err != nil {
		// Attempt to clean up the mount we just created
		runInHostMountNS("umount", target)
		return fmt.Errorf("failed to make mount read-only: %w", err)
	}

	return nil
}

// unmountInHostNS performs an unmount in the host's mount namespace.
// Uses nsenter to execute umount in the host namespace.
func unmountInHostNS(target string) error {
	if _, err := runInHostMountNS("umount", target); err != nil {
		return fmt.Errorf("failed to unmount %s: %w", target, err)
	}
	return nil
}

// tryUnmountInHostNS attempts to unmount in the host's mount namespace.
// Unlike unmountInHostNS, this doesn't return an error if the path wasn't mounted.
// This is useful for cleaning up potentially stale mounts that don't show in /proc/mounts.
func tryUnmountInHostNS(target string) error {
	output, err := runInHostMountNS("umount", target)
	if err != nil {
		// Check if the error is because it wasn't mounted (which is fine)
		outputStr := string(output)
		if contains(outputStr, "not mounted") || contains(outputStr, "no mount point") {
			return nil // Not an error - just wasn't mounted
		}
		return err
	}
	return nil
}

// contains checks if s contains substr (simple helper to avoid importing strings)
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// isPathMountedInHostNS checks if a path is mounted in the host's mount namespace.
// Uses nsenter to read /proc/mounts in the host namespace.
func isPathMountedInHostNS(targetPath string) (bool, error) {
	// First check if path exists in host namespace
	output, err := runInHostMountNS("test", "-e", targetPath)
	if err != nil {
		// test -e returns non-zero if path doesn't exist, which is not an error for us
		_ = output // ignore output
		return false, nil
	}

	// Read /proc/mounts in host namespace using cat
	output, err = runInHostMountNS("cat", "/proc/mounts")
	if err != nil {
		return false, fmt.Errorf("failed to read /proc/mounts: %w", err)
	}

	// Check if target is in mount list
	return isPathInMountOutput(targetPath, string(output)), nil
}

// isPathInMountOutput checks if a path is mounted based on /proc/mounts output
func isPathInMountOutput(targetPath, mountsOutput string) bool {
	for _, line := range splitLines(mountsOutput) {
		fields := splitFields(line)
		if len(fields) >= 2 {
			mountPoint := fields[1]
			if mountPoint == targetPath {
				return true
			}
		}
	}
	return false
}

// splitLines splits a string into lines (helper to avoid importing strings for minimal dependencies)
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// splitFields splits a string by whitespace (helper to avoid importing strings)
func splitFields(s string) []string {
	var fields []string
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			if start >= 0 {
				fields = append(fields, s[start:i])
				start = -1
			}
		} else {
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		fields = append(fields, s[start:])
	}
	return fields
}
