package versionsupport

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RuntimeVersionChecker interface {
	IsVersionSupported(version *version.Version) bool
	GetSupportedVersion() string
}

func IsRuntimeVersionSupported(ctx context.Context, details []v1alpha1.RuntimeDetailsByContainer) (bool, error) {
	logger := log.FromContext(ctx)

	for _, runtimeDetailsByContainer := range details {
		runtimeVersionSupporter := getRuntimeVersionCheck(runtimeDetailsByContainer.Language)
		if runtimeVersionSupporter == nil {
			logger.Info("Unsupported language", "language", runtimeDetailsByContainer.Language)
			return false, fmt.Errorf("Unsupported language: %s", runtimeDetailsByContainer.Language)
		}

		if runtimeDetailsByContainer.RuntimeVersion == "" {
			continue
		}

		runtimeVersion, err := version.NewVersion(runtimeDetailsByContainer.RuntimeVersion)
		if err != nil {
			logger.Info("Version format error: Invalid version for language",
				"runtimeVersion", runtimeDetailsByContainer.RuntimeVersion,
				"language", runtimeDetailsByContainer.Language,
			)
			return false, fmt.Errorf("Version format error: %s is not a valid version for language %s",
				runtimeDetailsByContainer.RuntimeVersion, runtimeDetailsByContainer.Language)
		}

		if !runtimeVersionSupporter.IsVersionSupported(runtimeVersion) {
			runtimeVersionOtelSDKSupport := runtimeVersionSupporter.GetSupportedVersion()
			logger.Info("Runtime version not supported by OpenTelemetry SDK",
				"language", runtimeDetailsByContainer.Language,
				"runtimeVersion", runtimeDetailsByContainer.RuntimeVersion,
				"supportedVersions", runtimeVersionOtelSDKSupport,
			)
			return false, fmt.Errorf("%s runtime version not supported by OpenTelemetry SDK. Found: %s, supports: %s",
				runtimeDetailsByContainer.Language, runtimeDetailsByContainer.RuntimeVersion, runtimeVersionOtelSDKSupport)
		}
	}

	return true, nil
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
	case common.UnknownProgrammingLanguage:
		return &UnknownVersionCheck{}
	case common.IgnoredProgrammingLanguage:
		return &IgnoredVersionCheck{}
	default:
		return nil
	}
}
