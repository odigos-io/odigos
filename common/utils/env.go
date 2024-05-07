package utils

import (
	"os"
	"path/filepath"

	"github.com/odigos-io/odigos/common/consts"

	"k8s.io/client-go/util/homedir"
)

const (
	KUBECONFIG = "KUBECONFIG"
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

func GetDefaultKubeConfigPath() string {
	if val, ok := os.LookupEnv(KUBECONFIG); ok {
		return val
	} else {
		if home := homedir.HomeDir(); home != "" {
			return filepath.Join(home, ".kube", "config")
		}
	}
	return ""
}
