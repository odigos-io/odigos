package collectorprofiles

import (
	"log"
	"os"
	"strings"
)

// profilingLogEnabled is true unless PROFILING_DEBUG=0 or false (profiling logs on by default for debugging deploy issues).
func profilingLogEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PROFILING_DEBUG")))
	return v != "0" && v != "false" && v != "off"
}

func profilingDebugLog(format string, args ...interface{}) {
	if !profilingLogEnabled() {
		return
	}
	log.Printf(format, args...)
}
