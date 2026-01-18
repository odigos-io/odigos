package helm

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

const (
	// DefaultCentralHelmChart is the default chart reference for Odigos Central.
	// It assumes the chart is published under the same repo as the OSS chart.
	DefaultCentralHelmChart = "odigos/odigos-central"

	// DefaultCentralReleaseName is the default Helm release name for Odigos Central.
	DefaultCentralReleaseName = "odigos-central"
)

type CentralValues struct {
	OnPremToken          string
	AdminUsername        string
	AdminPassword        string
	KeycloakStorageClass *string
	MaxMessageSize       string
	ImageTag             string
	ImagePullSecrets     []string
	ExternalOnpremSecret bool
}

// PrepareCentralChartAndValues loads the odigos-central Helm chart (embedded preferred)
// and returns the values map constructed from CentralValues.
func PrepareCentralChartAndValues(settings *cli.EnvSettings, chartVersion string, vals CentralValues) (*chart.Chart, map[string]interface{}, error) {
	version := strings.TrimPrefix(chartVersion, "v")

	// Prefer embedded chart for a matching version.
	ch, err := LoadEmbeddedChart(version, "odigos-central")
	if err == nil {
		fmt.Printf("📦 Using embedded chart %s (chart version: %s)\n", ch.Metadata.Name, ch.Metadata.Version)
		return ch, centralValuesToMap(vals), nil
	}

	// Fallback to repo chart.
	if err := ensureHelmRepo(settings, k8sconsts.OdigosHelmRepoName, k8sconsts.OdigosHelmRepoURL); err != nil {
		return nil, nil, err
	}
	refreshHelmRepo(settings, DefaultCentralHelmChart)

	chartPathOptions := action.ChartPathOptions{Version: version}
	chartPath, err := chartPathOptions.LocateChart(DefaultCentralHelmChart, settings)
	if err != nil {
		return nil, nil, err
	}

	ch, err = loader.Load(chartPath)
	if err != nil {
		return nil, nil, err
	}

	return ch, centralValuesToMap(vals), nil
}

func centralValuesToMap(v CentralValues) map[string]interface{} {
	out := map[string]interface{}{}

	if v.OnPremToken != "" {
		out["onPremToken"] = v.OnPremToken
	}
	if v.ExternalOnpremSecret {
		out["externalOnpremTokenSecret"] = true
	}

	if len(v.ImagePullSecrets) > 0 {
		out["imagePullSecrets"] = v.ImagePullSecrets
	}
	if v.ImageTag != "" {
		out["image"] = map[string]interface{}{"tag": v.ImageTag}
	}

	if v.MaxMessageSize != "" {
		out["centralBackend"] = map[string]interface{}{
			"maxMessageSize": v.MaxMessageSize,
		}
	}

	auth := map[string]interface{}{}
	if v.AdminUsername != "" {
		auth["adminUsername"] = v.AdminUsername
	}
	// Keep empty password as-is (chart will auto-generate on first install).
	auth["adminPassword"] = v.AdminPassword

	// Match the old CLI behavior: create PVC only when storageClassName is explicitly set and non-empty.
	if v.KeycloakStorageClass != nil && *v.KeycloakStorageClass != "" {
		auth["persistence"] = map[string]interface{}{
			"enabled":          true,
			"storageClassName": *v.KeycloakStorageClass,
		}
	}
	out["auth"] = auth

	return out
}
