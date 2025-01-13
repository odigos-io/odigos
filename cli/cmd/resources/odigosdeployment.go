package resources

import (
	"context"

	"github.com/odigos-io/odigos/api"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewOdigosDeploymentConfigMap(ns string, odigosVersion string, odigosTier string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      k8sconsts.OdigosDeploymentConfigMapName,
			Namespace: ns,
		},
		Data: map[string]string{
			k8sconsts.OdigosDeploymentConfigMapVersionKey:            odigosVersion,
			k8sconsts.OdigosDeploymentConfigMapTierKey:               odigosTier,
			k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey: string(installationmethod.K8sInstallationMethodOdigosCli),
		},
	}
}

func NewLeaderElectionRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odigos-leader-election-role",
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"update",
					"patch",
					"delete",
				},
				APIGroups: []string{""},
				Resources: []string{
					"configmaps",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"update",
					"patch",
					"delete",
				},
				APIGroups: []string{
					"coordination.k8s.io",
				},
				Resources: []string{
					"leases",
				},
			},
			{
				Verbs: []string{
					"create",
					"patch",
				},
				APIGroups: []string{""},
				Resources: []string{
					"events",
				},
			},
		},
	}
}

type odigosDeploymentResourceManager struct {
	client        *kube.Client
	ns            string
	config        *common.OdigosConfiguration
	odigosTier    common.OdigosTier
	odigosVersion string
}

func NewOdigosDeploymentResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosTier common.OdigosTier, odigosVersion string) resourcemanager.ResourceManager {
	return &odigosDeploymentResourceManager{client: client, ns: ns, config: config, odigosTier: odigosTier, odigosVersion: odigosVersion}
}

func (a *odigosDeploymentResourceManager) Name() string { return "OdigosDeployment" }

func (a *odigosDeploymentResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewOdigosDeploymentConfigMap(a.ns, a.odigosVersion, string(a.odigosTier)),
		NewLeaderElectionRole(a.ns),
	}

	excludedCRDs := []string{}
	availableCrds, err := api.GetCRDs(excludedCRDs)
	if err != nil {
		return err
	}

	for _, c := range availableCrds {
		resources = append(resources, c)
	}

	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
