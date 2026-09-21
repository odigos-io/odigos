package mergeconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeStringArrays_nilHandling(t *testing.T) {
	a := []string{"a"}

	require.Nil(t, MergeStringArrays(nil, nil))
	require.Equal(t, &a, MergeStringArrays(nil, &a))
	require.Equal(t, &a, MergeStringArrays(&a, nil))
}

func TestMergeStringArrays_dedupes(t *testing.T) {
	a1 := []string{"application/json", "text/plain"}
	a2 := []string{"text/plain", "application/xml"}

	got := MergeStringArrays(&a1, &a2)

	require.ElementsMatch(t, []string{"application/json", "text/plain", "application/xml"}, *got)
}

func TestMergeStringArrays_isDeterministic(t *testing.T) {
	// the merged value is persisted into the InstrumentationConfig spec, so an
	// unstable order rewrites the spec on every reconcile.
	first := *MergeStringArrays(&[]string{"d", "a"}, &[]string{"c", "b"})

	for i := 0; i < 100; i++ {
		got := *MergeStringArrays(&[]string{"d", "a"}, &[]string{"c", "b"})
		require.Equal(t, first, got)
	}
}

func TestMergeStringArrays_doesNotMutateInputs(t *testing.T) {
	a1 := []string{"b"}
	a2 := []string{"a"}

	MergeStringArrays(&a1, &a2)

	require.Equal(t, []string{"b"}, a1)
	require.Equal(t, []string{"a"}, a2)
}
