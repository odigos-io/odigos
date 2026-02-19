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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	response.InstallationStatus = getInstallationStatus(ctx)
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

// getInstallationStatus reads the persisted installation status from the
// odigos-local-ui-config ConfigMap. If not yet persisted, it falls back to
// computing the status from cluster state and persists the result for future calls.
func getInstallationStatus(ctx context.Context) model.InstallationStatus {
	status, err := readInstallationStatus(ctx)
	if err != nil {
		log.Printf("Error reading installation status: %v\n", err)
	}
	if status != "" {
		return model.InstallationStatus(status)
	}

	// Fallback: compute from cluster state (runs at most once per cluster lifetime)
	isNew := !isSourceCreated(ctx) && !isDestinationConnected(ctx)
	if isNew {
		return model.InstallationStatus(NewInstallation)
	}

	if err := persistInstallationStatus(ctx, string(Finished)); err != nil {
		log.Printf("Error persisting installation status: %v\n", err)
	}
	return model.InstallationStatus(Finished)
}

func readInstallationStatus(ctx context.Context) (string, error) {
	ns := env.GetCurrentNamespace()
	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosLocalUiConfigName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get local ui config: %w", err)
	}
	return cm.Data[k8sconsts.OdigosLocalUiInstallationStatusKey], nil
}

func persistInstallationStatus(ctx context.Context, status string) error {
	ns := env.GetCurrentNamespace()
	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosLocalUiConfigName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			ownerCm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get odigos-configuration for owner reference: %w", err)
			}

			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      consts.OdigosLocalUiConfigName,
					Namespace: ns,
					Labels: map[string]string{
						k8sconsts.OdigosSystemConfigLabelKey: "local-ui",
					},
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: "v1",
						Kind:       "ConfigMap",
						Name:       ownerCm.Name,
						UID:        ownerCm.UID,
					}},
				},
				Data: map[string]string{
					k8sconsts.OdigosLocalUiInstallationStatusKey: status,
				},
			}
			_, err = kube.DefaultClient.CoreV1().ConfigMaps(ns).Create(ctx, cm, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create local ui config ConfigMap: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get local ui config: %w", err)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[k8sconsts.OdigosLocalUiInstallationStatusKey] = status
	_, err = kube.DefaultClient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update local ui config ConfigMap: %w", err)
	}
	return nil
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
			for _, source := range sourceList.Items {
				if !source.Spec.DisableInstrumentation {
					return true
				}
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
