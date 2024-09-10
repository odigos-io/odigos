package versionsupport

import (
	"github.com/hashicorp/go-version"
	"strings"
)

type NginxVersionCheck struct{}

var nginxSupportedVersions = []string{"1.25.5", "1.26.0"}

func (g NginxVersionCheck) IsVersionSupported(version *version.Version) bool {
	for _, supportedVersion := range nginxSupportedVersions {
		if version.String() == supportedVersion {
			return true
		}
	}
	return false
}

func (g NginxVersionCheck) GetSupportedVersion() string {
	return strings.Join(nginxSupportedVersions, ", ")
}
