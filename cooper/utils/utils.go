package utils

import (
	"os"
)

const (
	CurrentNamespaceEnvVar = "CURRENT_NS"
	DefaultNamespace       = "odigos-system"
	CollectorLabel         = "odigos.io/collector"
	CommonConfigMapName    = "collector-conf"
)

func getEnvVarOrDefault(envKey string, defaultVal string) string {
	val, exists := os.LookupEnv(envKey)
	if exists {
		return val
	}

	return defaultVal
}

func GetCurrentNamespace() string {
	return getEnvVarOrDefault(CurrentNamespaceEnvVar, DefaultNamespace)
}
