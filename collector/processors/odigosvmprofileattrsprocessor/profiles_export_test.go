package odigosvmprofileattrsprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestProfilesExportable(t *testing.T) {
	require.False(t, profilesExportable(pprofile.NewProfiles()))

	profiles := pprofile.NewProfiles()
	profiles.ResourceProfiles().AppendEmpty()
	require.True(t, profilesExportable(profiles))
}
