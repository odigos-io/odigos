package utils

import (
	"fmt"

	"github.com/odigos-io/odigos/k8sutils/pkg/client"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
)

// ClusterVersion returns the Kubernetes control-plane version as a *version.Version.
func ClusterVersion() (*version.Version, error) {
	cfg, err := client.GetClientConfigWithContext(env.GetDefaultKubeConfigPath(), "")
	if err != nil {
		return nil, fmt.Errorf("build kube config: %w", err)
	}

	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}

	info, err := disco.ServerVersion() // simple helper; no ctx support
	if err != nil {
		return nil, fmt.Errorf("query /version: %w", err)
	}

	// Parse the **GitVersion** field, e.g. "v1.29.3+k3s2".
	v, err := version.ParseGeneric(info.GitVersion)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", info.GitVersion, err)
	}
	return v, nil
}
