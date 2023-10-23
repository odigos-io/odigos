package resources

import (
	"context"
	"encoding/json"

	"github.com/keyval-dev/odigos/cli/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	OdigosDeploymentConfigMapName = "odigos-deployment"
)

func NewOdigosDeploymentConfigMap(odigosVersion string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: OdigosDeploymentConfigMapName,
		},
		Data: map[string]string{
			"ODIGOS_VERSION": odigosVersion,
		},
	}
}

type odigosDeploymentResourceManager struct {
	client *kube.Client
	ns     string
}

func NewOdigosDeploymentResourceManager(client *kube.Client, ns string) ResourceManager {
	return &odigosDeploymentResourceManager{client: client, ns: ns}
}

func (a *odigosDeploymentResourceManager) InstallFromScratch(ctx context.Context) error {
	return nil
}

func (a *odigosDeploymentResourceManager) GetMigrationSteps() []MigrationStep {
	return []MigrationStep{}
}

// func (a *odigosDeploymentResourceManager) ApplyMigrationStep(ctx context.Context, sourceVersion string) error {
// 	return nil
// }

// func (a *odigosDeploymentResourceManager) RollbackMigrationStep(ctx context.Context, sourceVersion string) error {
// 	return nil
// }

func (a *odigosDeploymentResourceManager) PatchOdigosVersionToTarget(ctx context.Context, newOdigosVersion string) error {
	odigosVersionPatch := jsonPatchDocument{{
		Op:    "replace",
		Path:  "/data/ODIGOS_VERSION",
		Value: newOdigosVersion,
	}}

	jsonBytes, _ := json.Marshal(odigosVersionPatch)

	_, err := a.client.CoreV1().ConfigMaps(a.ns).Patch(ctx, OdigosDeploymentConfigMapName, k8stypes.JSONPatchType, jsonBytes, metav1.PatchOptions{})
	return err
}
