package opamp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
)

func TestResolveTransport(t *testing.T) {
	t.Parallel()

	// client disabled -> none regardless of supported transports
	require.Equal(t, OpAmpTransportNone,
		ResolveTransport(false, []OpAmpTransport{OpAmpTransportHTTP}, common.K8sVirtualDeviceMountMethod))

	// client enabled, empty supported -> defaults to http
	require.Equal(t, OpAmpTransportHTTP,
		ResolveTransport(true, nil, common.K8sInitContainerMountMethod))

	// http only -> http on any mount
	require.Equal(t, OpAmpTransportHTTP,
		ResolveTransport(true, []OpAmpTransport{OpAmpTransportHTTP}, common.K8sInitContainerMountMethod))

	// unix only on a mount that supports it -> unix
	require.Equal(t, OpAmpTransportUnix,
		ResolveTransport(true, []OpAmpTransport{OpAmpTransportUnix}, common.K8sVirtualDeviceMountMethod))

	// unix only on init-container -> none (no http fallback declared)
	require.Equal(t, OpAmpTransportNone,
		ResolveTransport(true, []OpAmpTransport{OpAmpTransportUnix}, common.K8sInitContainerMountMethod))

	// preference order [unix, http]: unix not usable on init-container -> http fallback
	require.Equal(t, OpAmpTransportHTTP,
		ResolveTransport(true, []OpAmpTransport{OpAmpTransportUnix, OpAmpTransportHTTP}, common.K8sInitContainerMountMethod))

	// preference order [unix, http]: unix usable -> unix wins
	require.Equal(t, OpAmpTransportUnix,
		ResolveTransport(true, []OpAmpTransport{OpAmpTransportUnix, OpAmpTransportHTTP}, common.K8sVirtualDeviceMountMethod))
}
