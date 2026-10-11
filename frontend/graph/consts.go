package graph

// Process attribute keys the agents emit on instrumentation instances.
// Used to give processes a stable, meaningful ordering across re-fetches
// without relying on the UI to sort them.
const (
	processAttributeNamePid  = "process.pid"
	processAttributeNameVpid = "process.vpid"
)
