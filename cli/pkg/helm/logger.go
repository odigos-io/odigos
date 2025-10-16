package helm

import (
	"fmt"
	"strings"
	"sync"
)

var (
	mu             sync.Mutex
	createdCount   int
	updatedCount   int
	deletedCount   int
	unchangedCount int
)

// CustomInstallLogger collects stats instead of printing every line based on helm install messages prefixes
func CustomInstallLogger(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)

	mu.Lock()
	defer mu.Unlock()

	switch {
	case strings.HasPrefix(msg, "creating"):
		createdCount++
	case strings.HasPrefix(msg, "Patch"):
		updatedCount++
	case strings.HasPrefix(msg, "Deleting"):
		deletedCount++
	case strings.HasPrefix(msg, "no changes"):
		unchangedCount++
	}
}

// CustomUninstallLogger collects stats instead of printing every line based on helm uninstall messages prefixes
func CustomUninstallLogger(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)

	mu.Lock()
	defer mu.Unlock()

	switch {
	case strings.HasPrefix(msg, "Starting delete"):
		deletedCount++
	}
}

// PrintSummary prints a concise summary at the end
func PrintSummary() {
	mu.Lock()
	defer mu.Unlock()

	fmt.Printf("\n⚡ Helm changes summary: %d created, %d updated, %d deleted, %d unchanged\n",
		createdCount, updatedCount, deletedCount, unchangedCount)

	// reset counters for next run
	createdCount, updatedCount, deletedCount, unchangedCount = 0, 0, 0, 0
}
