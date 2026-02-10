package env

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"

	"k8s.io/client-go/util/homedir"
)

const (
	KUBECONFIG                      = "KUBECONFIG"
	SYNC_DAEMONSET_DELAY_IN_SECONDS = "SYNC_DAEMONSET_DELAY_IN_SECONDS"
)

func getEnvVarOrDefault(envKey, defaultVal string) string {
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

// GetComponentDeploymentNameOrDefault returns the deployment name for this component.
// It reads ODIGOS_COMPONENT_DEPLOYMENT_NAME from the environment; if unset, returns defaultName.
func GetComponentDeploymentNameOrDefault(defaultName string) string {
	return getEnvVarOrDefault(consts.OdigosComponentDeploymentNameEnvVar, defaultName)
}

// GetOdigletDaemonSetNameOrDefault returns the odiglet DaemonSet name.
// It reads ODIGOS_ODIGLET_DAEMONSET_NAME from the environment; if unset, returns defaultName.
func GetOdigletDaemonSetNameOrDefault(defaultName string) string {
	return getEnvVarOrDefault(consts.OdigosOdigletDaemonSetNameEnvVar, defaultName)
}

// GetInstrumentorDeploymentNameOrDefault returns the instrumentor Deployment name.
// It reads ODIGOS_INSTRUMENTOR_DEPLOYMENT_NAME from the environment; if unset, returns defaultName.
func GetInstrumentorDeploymentNameOrDefault() string {
	return getEnvVarOrDefault(consts.OdigosInstrumentorDeploymentNameEnvVar, k8sconsts.InstrumentorDeploymentName)
}

func GetOdigosTierFromEnv() common.OdigosTier {
	odigosTierStr := os.Getenv(consts.OdigosTierEnvVarName)

	switch odigosTierStr {
	case string(common.CommunityOdigosTier):
		return common.CommunityOdigosTier
	case string(common.CloudOdigosTier):
		return common.CloudOdigosTier
	case string(common.OnPremOdigosTier):
		return common.OnPremOdigosTier
	default:
		return common.CommunityOdigosTier
	}
}
