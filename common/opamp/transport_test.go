package opamp

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestResolveTransport(t *testing.T) {
	t.Parallel()

	require.Equal(t, OpAmpTransportUnix, ResolveTransport(OpAmpTransportUnix, false, common.K8sVirtualDeviceMountMethod))
	require.Equal(t, OpAmpTransportHTTP, ResolveTransport(OpAmpTransportHTTP, false, common.K8sHostPathMountMethod))
	require.Equal(t, OpAmpTransportHTTP, ResolveTransport("", true, common.K8sVirtualDeviceMountMethod))
	require.Equal(t, OpAmpTransport(""), ResolveTransport("", false, common.K8sVirtualDeviceMountMethod))
	require.Equal(t, OpAmpTransport(""), ResolveTransport(OpAmpTransportUnix, false, common.K8sInitContainerMountMethod))
	require.Equal(t, OpAmpTransportHTTP, ResolveTransport("", true, common.K8sInitContainerMountMethod))
}

func TestParseOpAmpTransport(t *testing.T) {
	t.Parallel()

	transport, err := ParseOpAmpTransport("unix")
	require.NoError(t, err)
	require.Equal(t, OpAmpTransportUnix, transport)

	_, err = ParseOpAmpTransport("invalid")
	require.Error(t, err)
}
