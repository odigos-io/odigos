package opamp

import (
	"github.com/odigos-io/odigos/common"
)

// OpAmpTransport is the OpAMP client transport to inject into the workload.
type OpAmpTransport string

const (
	OpAmpTransportHTTP OpAmpTransport = "http"
	OpAmpTransportUnix OpAmpTransport = "unix"
	OpAmpTransportNone OpAmpTransport = "none" // do not inject ODIGOS_OPAMP_* env vars
)

// ResolveTransport picks which ODIGOS_OPAMP_* env var the webhook should inject.
//
// opAmpClientEnvironments is the intent flag (does this distro run an OpAMP client at all);
// supported is the ordered list of transports the distro's agent can speak. The first transport
// in the list that is usable on this node given the cluster constraints wins. An empty list
// defaults to [http] to preserve the historical behavior of distros that only set
// opAmpClientEnvironments: true.
func ResolveTransport(
	opAmpClientEnvironments bool,
	supported []OpAmpTransport,
	mountMethod common.MountMethod,
) OpAmpTransport {
	if !opAmpClientEnvironments {
		return OpAmpTransportNone
	}

	if len(supported) == 0 {
		supported = []OpAmpTransport{OpAmpTransportHTTP}
	}

	for _, t := range supported {
		if isTransportUsable(t, mountMethod) {
			return t
		}
	}
	return OpAmpTransportNone
}

func isTransportUsable(t OpAmpTransport, mountMethod common.MountMethod) bool {
	switch t {
	case OpAmpTransportHTTP:
		return true
	case OpAmpTransportUnix:
		// Unix socket cannot be reached from an init container that runs before odiglet's
		// /var/odigos mount is populated on the target pod.
		return mountMethod != common.K8sInitContainerMountMethod
	default:
		return false
	}
}
