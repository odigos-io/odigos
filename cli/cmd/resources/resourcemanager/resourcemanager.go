package resourcemanager

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ManagerOpts is used to pass options to each Resource Manager that installs Odigos resources.
type ManagerOpts struct {
	// ImageReferences refers to the images for each Odigos component.
	ImageReferences ImageReferences
	// OwnerReferences is a slice of metav1.OwnerReferences to be applied to each installed resource.
	OwnerReferences []metav1.OwnerReference
	// SystemObjectLabelScope defines which system label to use for Odigos resources (system-object or central-system-object).
	SystemObjectLabelKey string
	// NodeSelector is a Kubernetes NodeSelector that will be applied to all Odigos components.
	// Note that Odigos will only be able to instrument applications on the same node.
	NodeSelector map[string]string
}

type ImageReferences struct {
	AutoscalerImage     string
	CollectorImage      string
	InstrumentorImage   string
	OdigletImage        string
	KeyvalProxyImage    string
	SchedulerImage      string
	UIImage             string
	CentralProxyImage   string
	CentralBackendImage string
	CentralUIImage      string
}

type ResourceManager interface {
	Name() string

	// This function is being called to install the resource from scratch.
	// It should create all the required resources in the cluster, and return an error if the installation failed.
	// This function will only be invoked with `install`, thus it can assume that the resource is not installed in the cluster yet.
	// It is, however, preferable to make this function idempotent, so it can be invoked multiple times without causing any harm.
	InstallFromScratch(ctx context.Context) error
}
