package versionsupport

import "github.com/hashicorp/go-version"

type JavaVersionChecker struct{}

var JavaMinVersion, _ = version.NewVersion("17.0.11+8")

func (j JavaVersionChecker) IsVersionSupported(version *version.Version) bool {
	return true
}

func (j JavaVersionChecker) GetSupportedVersion() string {
	return JavaMinVersion.String()
}
