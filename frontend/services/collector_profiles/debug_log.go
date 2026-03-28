package collectorprofiles

import (
	"fmt"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

// profilingDebug logs verbose profiling pipeline details at debug severity (respects ODIGOS_LOG_LEVEL / commonlogger).
func profilingDebugLog(format string, args ...interface{}) {
	commonlogger.LoggerCompat().With("subsystem", "backend-profiling").Debug(fmt.Sprintf(format, args...))
}
