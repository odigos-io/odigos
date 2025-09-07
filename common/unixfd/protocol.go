package unixfd

// Protocol messages exchanged between odiglet (server) and data-collection (client).
const (
	// Client → Server
	ReqGetFD = "GET_FD"

	// Server → Client
	MsgNewFD  = "NEW_FD"  // sent only when a *new* map is created or odiglet restarts
	MsgFDSent = "FD_SENT" // sent in response to GET_FD when map already exists
)

const (
	DefaultSocketPath = "/var/exchange/exchange.sock"
)
