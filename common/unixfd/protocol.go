package unixfd

// Protocol messages exchanged between odiglet (server) and data-collection (client).
const (
	// Client → Server
	ReqGetFD = "GET_FD"

	// Server → Client
	RespOK = "OK"
)

const (
	DefaultSocketPath = "/var/exchange/exchange.sock"
)
