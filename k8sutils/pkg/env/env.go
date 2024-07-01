package env

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

// GetCurrentNamespace returns the namespace odigos is running in
func GetCurrentNamespace() string {
	return getEnvVarOrDefault(consts.CurrentNamespaceEnvVar, consts.DefaultOdigosNamespace)
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
