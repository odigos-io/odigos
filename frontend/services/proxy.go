package services

import (
    "context"
    "fmt"

    "github.com/odigos-io/odigos/api/k8sconsts"
    "github.com/odigos-io/odigos/common"
    "github.com/odigos-io/odigos/common/consts"
    "github.com/odigos-io/odigos/frontend/kube"
    "github.com/odigos-io/odigos/k8sutils/pkg/env"

    apierrors "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "sigs.k8s.io/yaml"
)

// DeleteCentralProxy deletes the central-proxy Deployment and clears central-backend URL from config
func DeleteCentralProxy(ctx context.Context) (bool, error) {
    ns := env.GetCurrentNamespace()

    // Delete deployment (ignore NotFound)
    if err := kube.DefaultClient.AppsV1().Deployments(ns).Delete(ctx, k8sconsts.CentralProxyDeploymentName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
        return false, fmt.Errorf("failed to delete central-proxy deployment: %v", err)
    }

    // Clear central-backend URL in odigos config
    cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{})
    if err != nil {
        return false, fmt.Errorf("failed to get odigos configuration: %v", err)
    }

    var cfg common.OdigosConfiguration
    if cm.Data != nil && cm.Data[consts.OdigosConfigurationFileName] != "" {
        if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &cfg); err != nil {
            cfg = common.OdigosConfiguration{}
        }
    }
    cfg.CentralBackendURL = ""

    yamlBytes, err := yaml.Marshal(cfg)
    if err != nil {
        return false, fmt.Errorf("failed to marshal odigos configuration: %v", err)
    }
    if cm.Data == nil {
        cm.Data = make(map[string]string)
    }
    cm.Data[consts.OdigosConfigurationFileName] = string(yamlBytes)

    if _, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
        return false, fmt.Errorf("failed to update odigos configuration: %v", err)
    }

    return true, nil
}


