package utils

import (
	"github.com/keyval-dev/odigos/common/consts"
	"os"
)

func getEnvVarOrDefault(envKey string, defaultVal string) string {
	val, exists := os.LookupEnv(envKey)
	if exists {
		return val
	}

	return defaultVal
}

func GetCurrentNamespace() string {
	return getEnvVarOrDefault(consts.CurrentNamespaceEnvVar, consts.DefaultNamespace)
}

func GetDetectorVersion() string {
	// TODO(edenfed): move this value to etcd as CRD, this may change during runtime
	return getEnvVarOrDefault(consts.LangDetectionEnvVar, consts.DefaultLangDetectionVersion)
}
