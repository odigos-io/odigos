package unixfd

// Protocol messages exchanged between odiglet (server) and data-collection (client).
const (
	// Client → Server
	ReqGetFD        = "GET_FD" // Legacy: defaults to traces for backward compatibility
	ReqGetTracesFD  = "GET_TRACES_FD"
	ReqGetMetricsFD = "GET_METRICS_FD"
	ReqGetLogsFD    = "GET_LOGS_FD"
	// ReqGetProfilesAttr requests a profiles attribute stream (no FD exchange).
	ReqGetProfilesAttr = "GET_PROF_ATTR"

	// Server → Client
	RespOK = "OK"
)

const (
	DefaultSocketPath = "/var/exchange/exchange.sock"
)
