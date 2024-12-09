package profiles

import "github.com/odigos-io/odigos/common"

func AgentsCanRunConcurrently(profiles []common.ProfileName) bool {
	for _, profile := range profiles {
		if profile == AllowConcurrentAgents.ProfileName {
			return true
		}

		profileDependencies := ProfilesMap[profile].Dependencies
		for _, dependencyProfile := range profileDependencies {
			if dependencyProfile == AllowConcurrentAgents.ProfileName {
				return true
			}
		}
	}
	return false
}

func FilterSizeProfiles(profiles []common.ProfileName) common.ProfileName {
	// In case multiple size profiles are provided, the first one will be used.
	for _, profile := range profiles {
		// Check if the profile is a size profile.
		switch profile {
		case SizeSProfile.ProfileName, SizeMProfile.ProfileName, SizeLProfile.ProfileName:
			return profile
		}

		// Check if the profile has a dependency which is a size profile.
		profileDependencies := ProfilesMap[profile].Dependencies
		for _, dependencyProfile := range profileDependencies {
			switch dependencyProfile {
			case SizeSProfile.ProfileName, SizeMProfile.ProfileName, SizeLProfile.ProfileName:
				return dependencyProfile
			}
		}
	}
	return ""
}
