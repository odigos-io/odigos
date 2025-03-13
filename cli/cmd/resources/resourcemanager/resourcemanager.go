package resourcemanager

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ManagerOpts struct {
	ImageReferences ImageReferences
	OwnerReferences []metav1.OwnerReference
}

type ImageReferences struct {
	AutoscalerImage   string
	CollectorImage    string
	InstrumentorImage string
	OdigletImage      string
	KeyvalProxyImage  string
	SchedulerImage    string
	UIImage           string
}

type ResourceManager interface {
	Name() string

	// This function is being called to install the resource from scratch.
	// It should create all the required resources in the cluster, and return an error if the installation failed.
	// This function will only be invoked with `install`, thus it can assume that the resource is not installed in the cluster yet.
	// It is, however, preferable to make this function idempotent, so it can be invoked multiple times without causing any harm.
	InstallFromScratch(ctx context.Context) error
}
