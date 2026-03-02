package unixfd

// Protocol messages exchanged between odiglet (server) and data-collection (client).
const (
	// Client → Server
	ReqGetFD        = "GET_FD" // Legacy: defaults to traces for backward compatibility
	ReqGetTracesFD  = "GET_TRACES_FD"
	ReqGetMetricsFD = "GET_METRICS_FD"
	ReqGetLogsFD    = "GET_LOGS_FD"

	// Server → Client
	RespOK = "OK"
)

const (
	DefaultSocketPath = "/var/exchange/exchange.sock"
)
