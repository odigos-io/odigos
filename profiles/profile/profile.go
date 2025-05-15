package profile

import (
	"github.com/odigos-io/odigos/common"
)

type Profile struct {
	ProfileName      common.ProfileName
	MinimumTier      common.OdigosTier
	ShortDescription string
	ManifestNames    []string                          // names of yaml manifest files to be applied by this profile
	Dependencies     []common.ProfileName              // other profiles that are applied by the current profile
	ModifyConfigFunc func(*common.OdigosConfiguration) // function to update the configuration based on the profile
}

func FindProfileByName(profileName common.ProfileName, profiles []Profile) *Profile {
	for i := range profiles {
		if profiles[i].ProfileName == profileName {
			return &profiles[i]
		}
	}
	return nil
}
