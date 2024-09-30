package versionsupport

import "github.com/hashicorp/go-version"

type PythonVersionCheck struct{}

var pythonMinVersion, _ = version.NewVersion("3.8")

func (g PythonVersionCheck) IsVersionSupported(version *version.Version) bool {
	return version.GreaterThanOrEqual(pythonMinVersion)
}

func (g PythonVersionCheck) GetSupportedVersion() string {
	return pythonMinVersion.String()
}
