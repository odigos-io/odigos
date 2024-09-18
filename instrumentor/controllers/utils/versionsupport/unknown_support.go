package versionsupport

import "github.com/hashicorp/go-version"

type UnknownVersionCheck struct{}

func (g UnknownVersionCheck) IsVersionSupported(version *version.Version) bool {
	// if we return false here, it will fail all device injection into all containers in the pod.
	// we return true here, which anyhow will not inject any device into the container.
	return true
}

func (g UnknownVersionCheck) GetSupportedVersion() string {
	return ""
}
