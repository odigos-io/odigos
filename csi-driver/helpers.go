package main

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
)

// checkRequiredPaths validates that all required host paths are accessible
// Returns true if all paths exist, false otherwise
func checkRequiredPaths() bool {
	requiredPaths := []string{
		KubeletDir,                // kubelet directory for CSI operations
		KubeletPluginsRegistryDir, // kubelet plugin registration
		OdigosAgentsDir,           // instrumentation files source
	}

	for _, path := range requiredPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			slog.Debug("Required path not accessible", "path", path)
			return false
		}
	}

	slog.Debug("All required paths accessible")
	return true
}

// extractPodUIDFromPath extracts just the pod UID from CSI target path for logging
// Expected format: /var/lib/kubelet/pods/{pod-uid}/volumes/kubernetes.io~csi/{volume-name}/mount
func extractPodUIDFromPath(targetPath string) string {
	// Simple regex to extract pod UID - much lighter than full parsing
	pattern := fmt.Sprintf(`%s/pods/([^/]+)/`, KubeletDir)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(targetPath)

	if len(matches) < 2 {
		return "unknown"
	}

	return matches[1]
}
