package versionsupport

import "github.com/hashicorp/go-version"

type GoVersionCheck struct{}

var goMinVersion, _ = version.NewVersion("1.17")

func (g GoVersionCheck) IsVersionSupported(version *version.Version) bool {
	return version.GreaterThanOrEqual(goMinVersion)
}

func (g GoVersionCheck) GetSupportedVersion() string {
	return goMinVersion.String()
}
