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
	var response model.GetConfigResponse

	response.Readonly = IsReadonlyMode(ctx)

	odigosDeployment, err := kube.DefaultClient.CoreV1().ConfigMaps(env.GetCurrentNamespace()).Get(ctx, k8sconsts.OdigosDeploymentConfigMapName, metav1.GetOptions{})
	if err != nil {
		response.Tier = model.Tier(common.CommunityOdigosTier)
	} else {
		response.Tier = model.Tier(odigosDeployment.Data[k8sconsts.OdigosDeploymentConfigMapTierKey])
	}

	if !isSourceCreated(ctx) && !isDestinationConnected(ctx) {
		response.Installation = model.InstallationStatus(NewInstallation)
	} else {
		response.Installation = model.InstallationStatus(Finished)
	}

	return response
}

func IsReadonlyMode(ctx context.Context) bool {
	ns := env.GetCurrentNamespace()

	configMap, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error getting config maps: %v\n", err)
		return false
	}

	var odigosConfiguration common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfiguration); err != nil {
		log.Printf("Error parsing YAML: %v\n", err)
		return false
	}

	return odigosConfiguration.UiMode == common.UiModeReadonly
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
