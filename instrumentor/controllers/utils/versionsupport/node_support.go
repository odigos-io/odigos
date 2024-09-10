package versionsupport

import "github.com/hashicorp/go-version"

type NodeVersionCheck struct{}

var nodeMinVersion, _ = version.NewVersion("14")

func (g NodeVersionCheck) IsVersionSupported(version *version.Version) bool {
	return version.GreaterThanOrEqual(nodeMinVersion)
}

func (g NodeVersionCheck) GetSupportedVersion() string {
	return nodeMinVersion.String()
}
