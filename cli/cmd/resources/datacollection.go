package resources

import (
	"context"

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
			Name:      "odigos-data-collection",
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
			Name: "odigos-data-collection",
		},
		Rules: []rbacv1.PolicyRule{
			{ // ??? ask for help, these are not regular methods that can be searched in-repo
				APIGroups: []string{""},
				Resources: []string{"endpoints", "nodes/stats", "nodes/proxy"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{ // Needed to get "resource name" in processor
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
			{ // Need "replicasets" to get "resource name" in processor,
				// Need others to get applications from cluster
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets", "deployments", "daemonsets", "statefulsets"},
				Verbs:     []string{"get", "list"},
			},
		},
	}

	if psp {
		clusterrole.Rules = append(clusterrole.Rules, rbacv1.PolicyRule{
			// ??? ask for help, these are not regular methods that can be searched in-repo
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
			Name: "odigos-data-collection",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "odigos-data-collection",
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "odigos-data-collection",
		},
	}
}

type dataCollectionResourceManager struct {
	client *kube.Client
	ns     string
	config *common.OdigosConfiguration
}

func NewDataCollectionResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration) resourcemanager.ResourceManager {
	return &dataCollectionResourceManager{client: client, ns: ns, config: config}
}

func (a *dataCollectionResourceManager) Name() string { return "DataCollection" }

func (a *dataCollectionResourceManager) InstallFromScratch(ctx context.Context) error {
	resources := []kube.Object{
		NewDataCollectionServiceAccount(a.ns),
		NewDataCollectionClusterRole(a.config.Psp),
		NewDataCollectionClusterRoleBinding(a.ns),
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
