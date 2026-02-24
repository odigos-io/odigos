package services

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type InstallationStatus string

const (
	NewInstallation InstallationStatus = "NEW"
	Finished        InstallationStatus = "FINISHED"
)

var (
	ErrorIsReadonly = errors.New("cannot execute this mutation in readonly mode")
)

func GetConfig(ctx context.Context) model.Config {
	odigosDeployment, err := kube.DefaultClient.CoreV1().ConfigMaps(env.GetCurrentNamespace()).Get(ctx, k8sconsts.OdigosDeploymentConfigMapName, metav1.GetOptions{})
	if err != nil {
		// assign default values (should not happen in production, but we want to be safe)
		odigosDeployment = &corev1.ConfigMap{}
		odigosDeployment.Data = map[string]string{
			k8sconsts.OdigosDeploymentConfigMapTierKey:               string(common.CommunityOdigosTier),
			k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey: string(installationmethod.K8sInstallationMethodOdigosCli),
		}
	}

	response := buildConfigResponse(ctx, odigosDeployment.Data)

	return response
}

func buildConfigResponse(ctx context.Context, deploymentData map[string]string) model.Config {
	var response model.Config
	config, err := GetOdigosConfiguration(ctx)
	if err != nil {
		log.Printf("Failed to get Config map: %v\n", err)
	}
	response.Readonly = config.UiMode == common.UiModeReadonly
	response.PlatformType = model.ComputePlatformTypeK8s
	response.Tier = model.Tier(deploymentData[k8sconsts.OdigosDeploymentConfigMapTierKey])
	response.OdigosVersion = deploymentData[k8sconsts.OdigosDeploymentConfigMapVersionKey]
	response.InstallationMethod = string(deploymentData[k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey])
	response.ClusterName = &config.ClusterName
	isConnected, err := isCentralProxyRunning(ctx)
	if err != nil {
		log.Printf("Error checking if central proxy connected: %v\n", err)
	}
	configured := isConfiguredForCentralBackend(common.OdigosTier(response.Tier), &config.ClusterName, config)
	if configured && isConnected {
		response.IsCentralProxyRunning = &isConnected
	} else {
		response.IsCentralProxyRunning = nil
	}
	response.InstallationStatus = getInstallationStatus(ctx, deploymentData)
	return response
}

func GetOdigosConfiguration(ctx context.Context) (*common.OdigosConfiguration, error) {
	ns := env.GetCurrentNamespace()

	configMap, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error getting config maps: %v\n", err)
		return nil, err
	}

	var odigosConfiguration common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfiguration); err != nil {
		log.Printf("Error parsing YAML from ConfigMap %s: %v\n", configMap.Name, err)
		return nil, err
	}

	return &odigosConfiguration, nil
}

func IsReadonlyMode(ctx context.Context) bool {
	config, err := GetOdigosConfiguration(ctx)
	if err != nil {
		return false
	}

	return config.UiMode == common.UiModeReadonly
}

func isConfiguredForCentralBackend(tier common.OdigosTier, clusterName *string, config *common.OdigosConfiguration) bool {
	if tier != common.OnPremOdigosTier {
		return false
	}

	if clusterName == nil || *clusterName == "" {
		return false
	}

	if config.CentralBackendURL == "" {
		return false
	}
	return true
}

func isCentralProxyRunning(ctx context.Context) (bool, error) {
	ns := env.GetCurrentNamespace()
	deployment, err := kube.DefaultClient.AppsV1().Deployments(ns).Get(ctx, k8sconsts.CentralProxyDeploymentName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Central proxy deployment not found: %v\n", err)
		return false, nil
	}
	if deployment.Status.AvailableReplicas == 0 {
		return false, nil
	}
	return true, nil
}

// getInstallationStatus reads the installation status from the already-fetched
// odigos-deployment ConfigMap data. If not yet persisted, it computes the status
// from cluster state and persists the result for future calls.
func getInstallationStatus(ctx context.Context, deploymentData map[string]string) model.InstallationStatus {
	if status := deploymentData[k8sconsts.OdigosDeploymentConfigMapInstallationStatusKey]; status != "" {
		return model.InstallationStatus(status)
	}

	// Compute from cluster state and persist the result
	computed := string(NewInstallation)
	if isSourceCreated(ctx) || isDestinationConnected(ctx) {
		computed = string(Finished)
	}

	if err := persistInstallationStatus(ctx, computed); err != nil {
		log.Printf("Error persisting installation status: %v\n", err)
	}
	return model.InstallationStatus(computed)
}

func persistInstallationStatus(ctx context.Context, status string) error {
	ns := env.GetCurrentNamespace()
	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, k8sconsts.OdigosDeploymentConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get odigos-deployment: %w", err)
	}
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[k8sconsts.OdigosDeploymentConfigMapInstallationStatusKey] = status
	_, err = kube.DefaultClient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

func isSourceCreated(ctx context.Context) bool {
	sourceList, err := kube.DefaultClient.OdigosClient.Sources("").List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {

		return false
	}

	return len(sourceList.Items) > 0
}

func isDestinationConnected(ctx context.Context) bool {
	ns := env.GetCurrentNamespace()

	dests, err := kube.DefaultClient.OdigosClient.Destinations(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing destinations: %v\n", err)
		return false
	}

	return len(dests.Items) > 0
}
