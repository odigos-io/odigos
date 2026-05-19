package opamp

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
)

// OpAmpTransport selects how an agent reaches the node-local OpAMP server.
type OpAmpTransport string

const (
	OpAmpTransportHTTP OpAmpTransport = "http"
	OpAmpTransportUnix OpAmpTransport = "unix"
)

// ResolveTransport decides which OpAMP transport env vars to inject for a distro.
// Returns empty when no OpAMP client env should be injected.
func ResolveTransport(
	opAmpTransport OpAmpTransport,
	opAmpClientEnvironments bool,
	mountMethod common.MountMethod,
) OpAmpTransport {
	if mountMethod == common.K8sInitContainerMountMethod {
		// Pod-local emptyDir cannot see the node socket.
		if opAmpClientEnvironments {
			return OpAmpTransportHTTP
		}
		return ""
	}

	switch opAmpTransport {
	case OpAmpTransportUnix:
		return OpAmpTransportUnix
	case OpAmpTransportHTTP:
		return OpAmpTransportHTTP
	}

	if opAmpClientEnvironments {
		return OpAmpTransportHTTP
	}
	return ""
}

// ParseOpAmpTransport parses a YAML or env value into OpAmpTransport.
func ParseOpAmpTransport(value string) (OpAmpTransport, error) {
	switch OpAmpTransport(value) {
	case "", OpAmpTransportHTTP, OpAmpTransportUnix:
		return OpAmpTransport(value), nil
	default:
		return "", fmt.Errorf("invalid opamp transport %q: must be %q or %q", value, OpAmpTransportHTTP, OpAmpTransportUnix)
	}
}
