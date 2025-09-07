package unixfd

// Protocol messages exchanged between odiglet (server) and data-collection (client).
// Can be extended in the future if needed.
const (
	// Client → Server
	ReqGetFD = "GET_FD"

	// Server → Client
	MsgNewFD  = "NEW_FD"  // server signals a new FD will be sent
	MsgFDSent = "FD_SENT" // server delivers the FD
)

const (
	DefaultSocketPath = "/var/exchange/exchange.sock"
)
