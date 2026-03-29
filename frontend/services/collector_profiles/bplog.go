package collectorprofiles

// bpInfof keeps old callsites but routes to the debug-gated logger.
func bpInfof(format string, args ...interface{}) {
	profilingDebugLog(format, args...)
}
