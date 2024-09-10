package versionsupport

import "github.com/hashicorp/go-version"

type MySQLVersionCheck struct{}

func (g MySQLVersionCheck) IsVersionSupported(version *version.Version) bool {
	return true
}

func (g MySQLVersionCheck) GetSupportedVersion() string {
	return "0.0.0"
}
