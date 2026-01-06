package services

import (
	"context"
	"errors"
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

func GetConfig(ctx context.Context) model.GetConfigResponse {
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

func buildConfigResponse(ctx context.Context, deploymentData map[string]string) model.GetConfigResponse {
	var response model.GetConfigResponse
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
	isNewInstallation := !isSourceCreated(ctx) && !isDestinationConnected(ctx)
	if isNewInstallation {
		response.InstallationStatus = model.InstallationStatus(NewInstallation)
	} else {
		response.InstallationStatus = model.InstallationStatus(Finished)
	}
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

func isSourceCreated(ctx context.Context) bool {
	ns := env.GetCurrentNamespace()

	nsList, err := getRelevantNameSpaces(ctx, ns)
	if err != nil {
		log.Printf("Error listing namespaces: %v\n", err)
		return false
	}

	for _, ns := range nsList {
		sourceList, err := kube.DefaultClient.OdigosClient.Sources(ns.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Error listing sources: %v\n", err)
			return false
		}

		if len(sourceList.Items) > 0 {
			allDisabled := true

			for _, source := range sourceList.Items {
				if !source.Spec.DisableInstrumentation {
					// Found an enabled source, no need to keep checking
					return true
				}
			}

			// If we get here, all sources were disabled
			if allDisabled {
				continue
			}
		}
	}

	return false
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
