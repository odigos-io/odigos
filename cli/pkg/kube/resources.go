package kube

import "k8s.io/apimachinery/pkg/runtime/schema"

func newApiExtensionsV1CustomResourceDefinition() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
}

func newAppsV1Deployments() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
}

func newAppsV1DaemonSet() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "daemonsets",
	}
}

func newCoreV1ConfigMaps() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "", // core group is represented by an empty string
		Version:  "v1",
		Resource: "configmaps",
	}
}

func deprecatedOdigosConfigs() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "odigos.io",
		Version:  "v1alpha1",
		Resource: "odigosconfigurations",
	}
}

func newCoreV1Services() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "", // core group is represented by an empty string
		Version:  "v1",
		Resource: "services",
	}
}

func newCoreV1Secrets() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "", // core group is represented by an empty string
		Version:  "v1",
		Resource: "secrets",
	}
}

func newRbacV1ClusterRoles() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "clusterroles",
	}
}

func newRbacV1ClusterRoleBindings() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "clusterrolebindings",
	}
}

func newRbacV1Roles() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "roles",
	}
}

func newRbacV1RoleBindings() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "rolebindings",
	}
}

func newInstrumentationRules() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "odigos.io",
		Version:  "v1alpha1",
		Resource: "instrumentationrules",
	}
}

func newRenameAttributeAction() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "actions.odigos.io",
		Version:  "v1alpha1",
		Resource: "renameattributes",
	}
}

func newProcessor() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "odigos.io",
		Version:  "v1alpha1",
		Resource: "processors",
	}
}

type ResourceAndNs struct {
	Resource  schema.GroupVersionResource
	Namespace string
}

func GetManagedResources(odigosNamespace string) []ResourceAndNs {
	return []ResourceAndNs{
		{Resource: newApiExtensionsV1CustomResourceDefinition(), Namespace: ""},
		{Resource: newAppsV1Deployments(), Namespace: odigosNamespace},
		{Resource: newAppsV1DaemonSet(), Namespace: odigosNamespace},
		{Resource: deprecatedOdigosConfigs(), Namespace: odigosNamespace},
		{Resource: newCoreV1ConfigMaps(), Namespace: odigosNamespace},
		{Resource: newCoreV1Services(), Namespace: odigosNamespace},
		{Resource: newCoreV1Secrets(), Namespace: odigosNamespace},
		{Resource: newRbacV1ClusterRoles(), Namespace: ""},
		{Resource: newRbacV1ClusterRoleBindings(), Namespace: ""},
		{Resource: newRbacV1Roles(), Namespace: odigosNamespace},
		{Resource: newRbacV1RoleBindings(), Namespace: odigosNamespace},
		{Resource: newInstrumentationRules(), Namespace: odigosNamespace},
		{Resource: newRenameAttributeAction(), Namespace: odigosNamespace},
		{Resource: newProcessor(), Namespace: odigosNamespace},
	}
}
