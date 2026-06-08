package unixfd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAttrEvent_WireFormatStable pins the on-the-wire encoding so the logs path (odiglet → collector)
// keeps working after the LogsAttr → AttrEvent rename. The receiver decodes the same bytes the publisher emits.
func TestAttrEvent_WireFormatStable(t *testing.T) {
	register := EncodeAttrRegister(123, "service.name:foo")
	require.Equal(t, "R123:service.name:foo", register)

	unreg := EncodeAttrUnregister(123)
	require.Equal(t, "U123", unreg)

	ev, ok := DecodeAttrEvent(register)
	require.True(t, ok)
	require.Equal(t, AttrEventRegister, ev.Type)
	require.Equal(t, uint32(123), ev.PID)
	require.Equal(t, "service.name:foo", ev.Attrs)

	ev, ok = DecodeAttrEvent(unreg)
	require.True(t, ok)
	require.Equal(t, AttrEventUnregister, ev.Type)
	require.Equal(t, uint32(123), ev.PID)
}

func TestAttrEvent_MalformedReturnsFalse(t *testing.T) {
	for _, in := range []string{"", "X", "R", "Rnotanint:foo", "U", "Unotanint", "R12"} {
		_, ok := DecodeAttrEvent(in)
		require.False(t, ok, "expected decode to fail for %q", in)
	}
}
