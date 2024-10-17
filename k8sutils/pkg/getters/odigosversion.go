package getters

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	ErrNoOdigosDeploymentConfigMap = errors.New("odigos deployment config map not found in cluster")
	ErrMissingVersionInConfigMap   = errors.New("odigos version not found in deployment config map")
)

// Return the Odigos version installed in the cluster.
// The function assumes odigos is installed, and will return an error if it is not or if the version cannot be detected (not expected in normal operation).
func GetOdigosVersionInClusterFromConfigMap(ctx context.Context, client kubernetes.Interface, ns string) (string, error) {
	cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosDeploymentConfigMapName, metav1.GetOptions{})
	if err != nil {
		return "", ErrNoOdigosDeploymentConfigMap
	}

	odigosVersion, ok := cm.Data["ODIGOS_VERSION"]
	if !ok || odigosVersion == "" {
		return "", ErrMissingVersionInConfigMap
	}

	return odigosVersion, nil
}
