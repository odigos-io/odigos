package versionsupport

import (
	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

type RuntimeVersionChecker interface {
	IsVersionSupported(version *version.Version) bool
}

func IsRuntimeVersionSupported(details []v1alpha1.RuntimeDetailsByContainer) bool {
	for _, runtimeDetailsByContainer := range details {

		runtimeVersionSupporter := getRuntimeVersionCheck(runtimeDetailsByContainer.Language)
		if runtimeVersionSupporter == nil {
			return false
		}

		if runtimeDetailsByContainer.RuntimeVersion == "" {
			// We haven't succeed to get the runtime version, so we can't check if it's supported
			return true
		}

		runtimeVersion, err := version.NewVersion(runtimeDetailsByContainer.RuntimeVersion)
		if err != nil {
			return false
		}

		if !runtimeVersionSupporter.IsVersionSupported(runtimeVersion) {
			return false
		}
	}

	return true
}

func getRuntimeVersionCheck(language common.ProgrammingLanguage) RuntimeVersionChecker {
	switch language {
	case common.JavaProgrammingLanguage:
		return &JavaVersionChecker{}
	case common.GoProgrammingLanguage:
		return &GoVersionCheck{}
	case common.PythonProgrammingLanguage:
		return &PythonVersionCheck{}
	case common.DotNetProgrammingLanguage:
		return &DotNetVersionCheck{}
	case common.JavascriptProgrammingLanguage:
		return &NodeVersionCheck{}
	case common.NginxProgrammingLanguage:
		return &NginxVersionCheck{}
	case common.MySQLProgrammingLanguage:
		return &MySQLVersionCheck{}
	default:
		return nil
	}
}
