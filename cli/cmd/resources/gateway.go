package resources

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewGatewayServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewGatewayRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorRoleName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{ // Needed to watch and retrieve configmaps that hold the collector config
				APIGroups:     []string{""},
				ResourceNames: []string{k8sconsts.OdigosClusterCollectorConfigMapName},
				Resources:     []string{"configmaps"},
				Verbs:         []string{"get", "list", "watch"},
			},
		},
	}
}

func NewGatewayRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.OdigosClusterCollectorServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     k8sconsts.OdigosClusterCollectorRoleName,
		},
	}
}

type gatewayResourceManager struct {
	client      *kube.Client
	ns          string
	config      *common.OdigosConfiguration
	managerOpts resourcemanager.ManagerOpts
}

func NewGatewayResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &gatewayResourceManager{client: client, ns: ns, config: config, managerOpts: managerOpts}
}

func (a *gatewayResourceManager) Name() string { return "Gateway" }

func (a *gatewayResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewGatewayServiceAccount(a.ns),
		NewGatewayRole(a.ns),
		NewGatewayRoleBinding(a.ns),
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
