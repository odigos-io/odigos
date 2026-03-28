package flamegraph

import "log"

const backendProfilingPrefix = "[backend-profiling]"

func bpFlamef(format string, args ...interface{}) {
	log.Printf(backendProfilingPrefix+" "+format, args...)
}
