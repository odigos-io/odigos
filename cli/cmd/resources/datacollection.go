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

func NewDataCollectionRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigletRoleName,
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

// NewDataCollectionConfigMap returns a safe "nop" OTEL config that lets the
// data-collection container start before the autoscaler/UI writes the real config.
func NewDataCollectionConfigMap(ns string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorConfigMapName,
			Namespace: ns,
			Labels: map[string]string{
				k8sconsts.OdigosPreserveLabelKey: "true",
			},
		},
		Data: map[string]string{
			// Minimal config: health_check + OTLP receiver to nop.
			// Its harmless and gets replaced by the controller later.
			"conf": `
extensions:
  health_check:
    endpoint: 0.0.0.0:13133

receivers:
  otlp:
    protocols:
      grpc:
      http:

exporters:
  nop: {}


service:
  extensions: [health_check]
  pipelines:
    traces:
      receivers: [otlp]
      processors: []
      exporters: [nop]
`,
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
	}

	// Create-once bootstrap CM (only if missing)
	bootstrapCM := NewDataCollectionConfigMap(a.ns)
	if err := a.client.ApplyResourceIfAbsent(ctx, a.config.ConfigVersion, bootstrapCM, a.managerOpts); err != nil {
		return err
	}

	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources, a.managerOpts)
}
