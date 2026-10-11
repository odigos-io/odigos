package k8sconsts

const (
	OdigosAgentsDirectory          = "/var/odigos"
	OdigletContainerAgentDirectory = "/instrumentations"
	OdigosAgentMountVolumeName     = "odigos-agent"

	OdigosOpampExchangeDir    = "/var/odigos/exchange"
	OdigosOpampUnixSocketPath = "/var/odigos/exchange/exchange.sock"

	OpampUnixSocketEnvName = "ODIGOS_OPAMP_UNIX_SOCKET"
)
