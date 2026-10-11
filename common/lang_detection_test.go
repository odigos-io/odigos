package common

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/require"
)

func TestParseRuntimeVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		wantNil  bool
		wantCore string
	}{
		{input: "1.22.0", wantCore: "1.22.0"},
		{input: "v18.0.0", wantCore: "18.0.0"},
		{input: "v1.2.3-0", wantCore: "1.2.3"},
		{input: "1.2.3-0", wantCore: "1.2.3"},
		{input: "1.2.3+build", wantCore: "1.2.3"},
		{input: "not-a-version", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := ParseRuntimeVersion(tt.input)
			if tt.wantNil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tt.wantCore, got.String())
		})
	}
}

func TestParseRuntimeVersion_satisfiesConstraintWithPrereleaseSuffix(t *testing.T) {
	t.Parallel()

	v := ParseRuntimeVersion("v1.22.0-0")
	require.NotNil(t, v)

	constraint, err := version.NewConstraint(">= 1.19")
	require.NoError(t, err)
	require.True(t, constraint.Check(v))
}
