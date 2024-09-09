package versionsupport

import "github.com/hashicorp/go-version"

type DotNetVersionCheck struct{}

func (g DotNetVersionCheck) IsVersionSupported(version *version.Version) bool {
	return true
}
