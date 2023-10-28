package resources

import (
	"context"

	"github.com/keyval-dev/odigos/cli/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func NewLeaderElectionRole() *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-leader-election-role",
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
	client  *kube.Client
	ns      string
	version string
}

func NewOdigosDeploymentResourceManager(client *kube.Client, ns string, version string) ResourceManager {
	return &odigosDeploymentResourceManager{client: client, ns: ns, version: version}
}

func (a *odigosDeploymentResourceManager) Name() string { return "OdigosDeployment" }

func (a *odigosDeploymentResourceManager) InstallFromScratch(ctx context.Context) error {
	cm := NewOdigosDeploymentConfigMap(a.version)
	err := a.client.ApplyResource(ctx, a.ns, a.version, cm)
	if err != nil {
		return err
	}

	role := NewLeaderElectionRole()
	err = a.client.ApplyResource(ctx, a.ns, a.version, role)
	if err != nil {
		return err
	}

	return nil
}
