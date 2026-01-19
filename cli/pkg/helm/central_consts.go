package helm

import "github.com/odigos-io/odigos/api/k8sconsts"

const (
	// DefaultCentralHelmChart is the default chart reference for Odigos Central.
	// Keep this in sync with k8sconsts.DefaultCentralHelmChart.
	DefaultCentralHelmChart = k8sconsts.DefaultCentralHelmChart

	// DefaultCentralReleaseName is the default Helm release name for Odigos Central.
	DefaultCentralReleaseName = "odigos-central"
)
