package versionsupport

import "github.com/hashicorp/go-version"

type IgnoredVersionCheck struct{}

func (g IgnoredVersionCheck) IsVersionSupported(version *version.Version) bool {
	// ignored containers are anyhow not used for device injection.
	// we will return true here so not to fail all containers due to this check.
	return true
}

func (g IgnoredVersionCheck) GetSupportedVersion() string {
	return ""
}
