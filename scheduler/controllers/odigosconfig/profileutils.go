package odigosconfig

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

// when applying the manifests, we need to mark the new resources so we know to cleanup
// previous resources which are not relevant after the new profiles are applied.
// this hash is used to mark the resources.
func calculateProfilesDeploymentHash(profiles []common.ProfileName, odigosVersion string) string {

	// convert profiles to a list of strings and sort it for consistency
	sortedProfiles := make([]string, len(profiles)+1)
	for i, p := range profiles {
		sortedProfiles[i] = string(p)
	}
	sortedProfiles[len(profiles)] = odigosVersion
	sort.Strings(sortedProfiles)

	// Concatenate the sorted strings to create a hash
	concatenated := strings.Join(sortedProfiles, "")
	hash := sha256.Sum256([]byte(concatenated))

	// Return the hash as a hex string
	return hex.EncodeToString(hash[:16])
}

// from the list of input profiles, calculate the effective profiles:
// - check the dependencies of each profile and add them to the list
// - remove profiles which are not present in the profiles list
func calculateEffectiveProfiles(configProfiles []common.ProfileName, availableProfiles []profile.Profile) []common.ProfileName {

	effectiveProfiles := []common.ProfileName{}
	for _, profileName := range configProfiles {

		// ignored missing profiles (either not available for tier or typos)
		p, found := findProfileNameInAvailableList(profileName, availableProfiles)
		if !found {
			continue
		}

		effectiveProfiles = append(effectiveProfiles, profileName)

		// if this profile has dependencies, add them to the list
		if p.Dependencies != nil {
			effectiveProfiles = append(effectiveProfiles, calculateEffectiveProfiles(p.Dependencies, availableProfiles)...)
		}
	}
	return effectiveProfiles
}

func findProfileNameInAvailableList(profileName common.ProfileName, availableProfiles []profile.Profile) (profile.Profile, bool) {
	// there aren't many profiles, so a linear search is fine
	for _, p := range availableProfiles {
		if p.ProfileName == profileName {
			return p, true
		}
	}
	return profile.Profile{}, false
}
