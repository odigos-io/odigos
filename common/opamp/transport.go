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
// When opAmpTransport is empty, opAmpClientEnvironments defaults to http.
func ResolveTransport(opAmpTransport OpAmpTransport, opAmpClientEnvironments bool, mountMethod common.MountMethod, runtimeVersion string) OpAmpTransport {
	if !opAmpClientEnvironments {
		return OpAmpTransportNone
	}

	transport := OpAmpTransportHTTP
	if opAmpTransport == OpAmpTransportUnix {
		transport = OpAmpTransportUnix
	}

	if transport == OpAmpTransportUnix {
		if mountMethod == common.K8sInitContainerMountMethod {
			return OpAmpTransportNone
		}
		if !javaSupportsOpAmpUnix(runtimeVersion) {
			return OpAmpTransportNone
		}
	}

	return transport
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
