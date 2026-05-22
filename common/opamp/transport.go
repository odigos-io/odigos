package opamp

import (
	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
)

// OpAmpTransport is the OpAMP client transport to inject into the workload.
type OpAmpTransport string

const (
	OpAmpTransportHTTP OpAmpTransport = "http"
	OpAmpTransportUnix OpAmpTransport = "unix"
	OpAmpTransportNone OpAmpTransport = "none" // do not inject ODIGOS_OPAMP_* env vars

	// java unix OpAMP needs JVM 16+ (java.net Unix sockets / SO_RCVTIMEO in the agent).
	javaOpAmpUnixMinVersion = ">= 16.0.0"
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
	runtimeVersion string,
) OpAmpTransport {
	if !opAmpClientEnvironments {
		return OpAmpTransportNone
	}

	if len(supported) == 0 {
		supported = []OpAmpTransport{OpAmpTransportHTTP}
	}

	for _, t := range supported {
		if isTransportUsable(t, mountMethod, runtimeVersion) {
			return t
		}
	}
	return OpAmpTransportNone
}

// checks if the transport is usable on given constraints (mount method, runtime version).
func isTransportUsable(t OpAmpTransport, mountMethod common.MountMethod, runtimeVersion string) bool {
	switch t {
	case OpAmpTransportHTTP:
		// http is always usable regardless of mount method or runtime version
		return true
	case OpAmpTransportUnix:
		if mountMethod == common.K8sInitContainerMountMethod {
			return false
		}
		return javaSupportsOpAmpUnix(runtimeVersion)
	default:
		return false
	}
}

func javaSupportsOpAmpUnix(runtimeVersion string) bool {
	v := common.GetVersion(runtimeVersion)
	if v == nil {
		return false
	}
	constraint, err := version.NewConstraint(javaOpAmpUnixMinVersion)
	if err != nil {
		return false
	}
	return constraint.Check(v)
}
