package env

import (
	"github.com/odigos-io/odigos/common/consts"
	"os"
	"path/filepath"
	"strconv"

	"k8s.io/client-go/util/homedir"
)

const (
	KUBECONFIG                      = "KUBECONFIG"
	SYNC_DAEMONSET_DELAY_IN_SECONDS = "SYNC_DAEMONSET_DELAY_IN_SECONDS"
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

func GetSyncDaemonSetDelay() int {
	delay := getEnvVarOrDefault(SYNC_DAEMONSET_DELAY_IN_SECONDS, "5")
	delayValue, err := strconv.Atoi(delay)
	if err != nil {
		return 5
	}

	return delayValue
}
