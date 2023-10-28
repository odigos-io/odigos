package resources

import (
	"context"

	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/keyval-dev/odigos/cli/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDataCollectionServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "odigos-data-collection",
			Labels: labels.OdigosSystem,
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
			Name:   "odigos-data-collection",
			Labels: labels.OdigosSystem,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{""},
				Resources: []string{
					"pods",
					"nodes/stats",
					"nodes/proxy",
				},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
				},
				APIGroups: []string{"apps"},
				Resources: []string{
					"replicasets",
					"deployments",
					"daemonsets",
					"statefulsets",
				},
			},
		},
	}

	if psp {
		clusterrole.Rules = append(clusterrole.Rules, rbacv1.PolicyRule{
			Verbs: []string{
				"use",
			},
			APIGroups: []string{
				"policy",
			},
			Resources: []string{
				"podsecuritypolicies",
			},
			ResourceNames: []string{
				"privileged",
			},
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
			Name:   "odigos-data-collection",
			Labels: labels.OdigosSystem,
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
	client  *kube.Client
	ns      string
	version string
	psp     bool
}

func NewDataCollectionResourceManager(client *kube.Client, ns string, version string, psp bool) ResourceManager {
	return &dataCollectionResourceManager{client: client, ns: ns, version: version, psp: psp}
}

func (a *dataCollectionResourceManager) Name() string { return "DataCollection" }

func (a *dataCollectionResourceManager) InstallFromScratch(ctx context.Context) error {

	sa := NewDataCollectionServiceAccount()
	err := a.client.ApplyResource(ctx, a.ns, sa, sa.TypeMeta, sa.ObjectMeta)
	if err != nil {
		return err
	}

	clusterRole := NewDataCollectionClusterRole(a.psp)
	err = a.client.ApplyResource(ctx, "", clusterRole, clusterRole.TypeMeta, clusterRole.ObjectMeta)
	if err != nil {
		return err
	}

	clusterRoleBinding := NewDataCollectionClusterRoleBinding(a.ns)
	err = a.client.ApplyResource(ctx, "", clusterRoleBinding, clusterRoleBinding.TypeMeta, clusterRoleBinding.ObjectMeta)
	if err != nil {
		return err
	}

	return nil
}
