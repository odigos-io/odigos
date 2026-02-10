package services

import (
	"context"
	"errors"
	"log"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	var odigosDeployment corev1.ConfigMap
	err := kube.CacheClient.Get(ctx, ctrlclient.ObjectKey{
		Namespace: env.GetCurrentNamespace(),
		Name:      k8sconsts.OdigosDeploymentConfigMapName,
	}, &odigosDeployment)
	if err != nil {
		// assign default values (should not happen in production, but we want to be safe)
		odigosDeployment.Data = map[string]string{
			k8sconsts.OdigosDeploymentConfigMapTierKey:               string(common.CommunityOdigosTier),
			k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey: string(installationmethod.K8sInstallationMethodOdigosCli),
		}
	}

	return buildConfigResponse(ctx, odigosDeployment.Data)
}

func buildConfigResponse(ctx context.Context, deploymentData map[string]string) model.Config {
	config, err := GetOdigosConfiguration(ctx)
	if err != nil {
		log.Printf("Failed to get Config map: %v\n", err)
	}
	if config == nil {
		config = &common.OdigosConfiguration{}
	}

	var response model.Config
	response.Readonly = config.UiMode == common.UiModeReadonly
	response.PlatformType = model.ComputePlatformTypeK8s
	response.Tier = model.Tier(deploymentData[k8sconsts.OdigosDeploymentConfigMapTierKey])
	response.OdigosVersion = deploymentData[k8sconsts.OdigosDeploymentConfigMapVersionKey]
	response.InstallationMethod = string(deploymentData[k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey])
	response.ClusterName = &config.ClusterName

	isConnected := isCentralProxyRunning(ctx)
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
	var configMap corev1.ConfigMap
	err := kube.CacheClient.Get(ctx, ctrlclient.ObjectKey{
		Namespace: env.GetCurrentNamespace(),
		Name:      consts.OdigosEffectiveConfigName,
	}, &configMap)
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

func isCentralProxyRunning(ctx context.Context) bool {
	ns := env.GetCurrentNamespace()
	var deployment appsv1.Deployment
	err := kube.CacheClient.Get(ctx, ctrlclient.ObjectKey{
		Namespace: ns,
		Name:      k8sconsts.CentralProxyDeploymentName,
	}, &deployment)
	if err != nil {
		return false
	}
	return deployment.Status.AvailableReplicas > 0
}

func isSourceCreated(ctx context.Context) bool {
	var sourceList odigosv1.SourceList
	if err := kube.CacheClient.List(ctx, &sourceList); err != nil {
		log.Printf("Error listing sources from cache: %v\n", err)
		return false
	}

	for _, source := range sourceList.Items {
		if !source.Spec.DisableInstrumentation {
			return true
		}
	}

	return false
}

func isDestinationConnected(ctx context.Context) bool {
	ns := env.GetCurrentNamespace()

	dests, err := kube.DefaultClient.OdigosClient.Destinations(ns).List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		log.Printf("Error listing destinations: %v\n", err)
		return false
	}

	return len(dests.Items) > 0
}
