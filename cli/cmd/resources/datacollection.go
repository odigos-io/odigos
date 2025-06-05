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

func NewDataCollectionServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorServiceAccountName,
			Namespace: ns,
		},
	}
}

func NewDataCollectionClusterRole(psp bool) *rbacv1.ClusterRole {
	clusterrole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.OdigosNodeCollectorClusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{ // TODO: remove this after we remove honeycomb custom exporter config
				// located at: autoscaler/controllers/datacollection/custom/honeycomb.go
				APIGroups: []string{""},
				Resources: []string{"nodes/stats", "nodes/proxy"},
				Verbs:     []string{"get", "list"},
			},
			{ // Need for k8s attributes processor
				APIGroups: []string{""},
				Resources: []string{"pods", "namespaces"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Need for k8s attributes processor
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets", "deployments", "daemonsets", "statefulsets"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed for load balancer
				APIGroups: []string{""},
				Resources: []string{"endpoints"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	if psp {
		clusterrole.Rules = append(clusterrole.Rules, rbacv1.PolicyRule{
			// Needed for clients who enable pod security policies
			APIGroups:     []string{"policy"},
			Resources:     []string{"podsecuritypolicies"},
			ResourceNames: []string{"privileged"},
			Verbs:         []string{"use"},
		})
	}

	return clusterrole
}

func NewDataCollectionClusterRoleBinding(ns string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.OdigosNodeCollectorClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.OdigosNodeCollectorServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     k8sconsts.OdigosNodeCollectorClusterRoleName,
		},
	}
}

func NewDataCollectionRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorRoleName,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{ // Needed for configmap provider to watch for config updates inside the collector
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{k8sconsts.OdigosNodeCollectorConfigMapName},
				Verbs:         []string{"get", "list", "watch"},
			},
		},
	}
}

func NewDataCollectionRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorRoleBindingName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      k8sconsts.OdigosNodeCollectorServiceAccountName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     k8sconsts.OdigosNodeCollectorRoleName,
		},
	}
}

type dataCollectionResourceManager struct {
	client      *kube.Client
	ns          string
	config      *common.OdigosConfiguration
	managerOpts resourcemanager.ManagerOpts
}

func NewDataCollectionResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, managerOpts resourcemanager.ManagerOpts) resourcemanager.ResourceManager {
	return &dataCollectionResourceManager{client: client, ns: ns, config: config, managerOpts: managerOpts}
}

func (a *dataCollectionResourceManager) Name() string { return "DataCollection" }

func (a *dataCollectionResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewDataCollectionServiceAccount(a.ns),
		NewDataCollectionRole(a.ns),
		NewDataCollectionRoleBinding(a.ns),
		NewDataCollectionClusterRole(a.config.Psp),
		NewDataCollectionClusterRoleBinding(a.ns),
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
