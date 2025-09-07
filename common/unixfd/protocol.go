package unixfd

// Currently we only need to request a single FD (ReqGetFD).
// In the future this can be extended with more messages types (e.g "FD_GET_TRACES", "FD_GET_METRICS", "PING", etc).
// Keeping them as constants here makes both server and client share the same protocol.
const (
	ReqGetFD = "GET_FD"
)

const (
	DefaultSocketPath = "/var/exchange/exchange.sock"
)
