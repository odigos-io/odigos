package helm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	HelmReleaseName  string
	HelmNamespace    string
	HelmChart        string
	HelmValuesFile   string
	HelmSetArgs      []string
	HelmChartVersion string
)

// injected at build time
var OdigosChartVersion string

var (
	HelmResetThenReuseValues = true // default: true (sensible for upgrades)
)

func PrepareChartAndValues(settings *cli.EnvSettings, chartName string) (*chart.Chart, map[string]interface{}, error) {
	return prepareChartAndValues(settings, chartName, k8sconsts.DefaultHelmChart)
}

func PrepareCentralChartAndValues(settings *cli.EnvSettings, chartName string) (*chart.Chart, map[string]interface{}, error) {
	return prepareChartAndValues(settings, chartName, k8sconsts.DefaultCentralHelmChart)
}

// prepareChartAndValues is the common implementation for both OSS and Central flows.
// - chartName controls which embedded chart archive to load (e.g. "odigos" / "odigos-central")
// - embeddedGateChart controls when we attempt embedded chart first (i.e. when HelmChart == embeddedGateChart and no --chart-version override)
func prepareChartAndValues(settings *cli.EnvSettings, chartName string, embeddedGateChart string) (*chart.Chart, map[string]interface{}, error) {
	version := ""
	if HelmChartVersion != "" {
		version = strings.TrimPrefix(HelmChartVersion, "v")
	} else if OdigosChartVersion != "" {
		version = strings.TrimPrefix(OdigosChartVersion, "v")
	}
	// Use embedded chart if available (when using the default chart and no override)
	if HelmChart == embeddedGateChart && HelmChartVersion == "" {
		ch, err := LoadEmbeddedChart(version, chartName)
		if err == nil {
			fmt.Printf("📦 Using embedded chart %s (chart version: %s)\n", ch.Metadata.Name, ch.Metadata.Version)

			// merge values like normal (so --set and --values flags work)
			valOpts := &values.Options{
				ValueFiles: []string{},
				Values:     HelmSetArgs,
			}
			if HelmValuesFile != "" {
				valOpts.ValueFiles = append(valOpts.ValueFiles, HelmValuesFile)
			}
			vals, err := valOpts.MergeValues(getter.All(settings))
			if err != nil {
				return nil, nil, err
			}

			// fallback image.tag to AppVersion if not set
			// During the release of the helm chart, we're setting the appVersion to the same as the image.tag [package-charts.sh]
			if ch.Metadata.AppVersion != "" {
				if _, ok := vals["image"]; !ok {
					vals["image"] = map[string]interface{}{}
				}
				if imgVals, ok := vals["image"].(map[string]interface{}); ok {
					if _, hasTag := imgVals["tag"]; !hasTag || imgVals["tag"] == "" {
						imgVals["tag"] = ch.Metadata.AppVersion
						fmt.Printf("Using the Chart appVersion %s as image.tag\n", ch.Metadata.AppVersion)
					}
				}
			}

			return ch, vals, nil
		}
		// if no embedded chart found, continue with repo fallback
	}

	// otherwise: use remote/local chart like today
	if strings.HasPrefix(HelmChart, k8sconsts.OdigosHelmRepoName+"/") {
		if err := ensureHelmRepo(settings, k8sconsts.OdigosHelmRepoName, k8sconsts.OdigosHelmRepoURL); err != nil {
			return nil, nil, err
		}
	}
	if strings.HasPrefix(HelmChart, k8sconsts.OdigosCentralHelmRepoName+"/") {
		fmt.Println("Ensuring Odigos Central Helm repo")
		if err := ensureHelmRepo(settings, k8sconsts.OdigosCentralHelmRepoName, k8sconsts.OdigosCentralHelmRepoURL); err != nil {
			return nil, nil, err
		}
	}
	if strings.Contains(HelmChart, "/") {
		refreshHelmRepo(settings, HelmChart)
	}

	chartPathOptions := action.ChartPathOptions{Version: version}
	chartPath, err := chartPathOptions.LocateChart(HelmChart, settings)
	if err != nil {
		return nil, nil, err
	}
	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, nil, err
	}

	valOpts := &values.Options{
		ValueFiles: []string{},
		Values:     HelmSetArgs,
	}
	if HelmValuesFile != "" {
		valOpts.ValueFiles = append(valOpts.ValueFiles, HelmValuesFile)
	}
	vals, err := valOpts.MergeValues(getter.All(settings))
	if err != nil {
		return nil, nil, err
	}

	if ch.Metadata.AppVersion != "" {
		if _, ok := vals["image"]; !ok {
			vals["image"] = map[string]interface{}{}
		}
		if imgVals, ok := vals["image"].(map[string]interface{}); ok {
			if _, hasTag := imgVals["tag"]; !hasTag || imgVals["tag"] == "" {
				imgVals["tag"] = ch.Metadata.AppVersion
				fmt.Printf("Using appVersion %s as image.tag\n", ch.Metadata.AppVersion)
			}
		}
	}

	return ch, vals, nil
}

// ensureHelmRepo adds a repo if missing
func ensureHelmRepo(settings *cli.EnvSettings, name, url string) error {
	repoFile := settings.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	// Use errors.Is to properly handle wrapped errors from Helm
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// check if repo already exists
	if f != nil {
		for _, r := range f.Repositories {
			if r.Name == name {
				return nil // already present
			}
		}
	} else {
		f = repo.NewFile()
	}

	// add new repo
	entry := &repo.Entry{Name: name, URL: url}
	r, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		return err
	}
	_, err = r.DownloadIndexFile()
	if err != nil {
		return fmt.Errorf("cannot reach repo %s at %s: %w", name, url, err)
	}
	f.Update(entry)
	return f.WriteFile(repoFile, 0644)
}

// refreshHelmRepo updates repo index (like `helm repo update`)
func refreshHelmRepo(settings *cli.EnvSettings, chartRef string) {
	repoFile := settings.RepositoryConfig
	repoCache := settings.RepositoryCache

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		fmt.Printf("Warning: cannot load Helm repo file: %v\n", err)
		return
	}

	for _, r := range f.Repositories {
		if strings.HasPrefix(chartRef, r.Name+"/") {
			chartRepo, err := repo.NewChartRepository(r, getter.All(settings))
			if err != nil {
				fmt.Printf("Warning: cannot create repo client for %s: %v\n", r.Name, err)
				continue
			}
			chartRepo.CachePath = repoCache
			_, err = chartRepo.DownloadIndexFile()
			if err != nil {
				fmt.Printf("Warning: failed to update repo %s: %v\n", r.Name, err)
			} else {
				fmt.Printf("Updated Helm repo: %s\n", r.Name)
			}
		}
	}
}

// IsLegacyInstallation checks whether Odigos was installed using the old non-Helm method.
func IsLegacyInstallation(ctx context.Context, client corev1.CoreV1Interface, namespace string) (bool, error) {
	cm, err := client.ConfigMaps(namespace).Get(ctx, k8sconsts.OdigosDeploymentConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// ConfigMap doesn’t exist — not an old install
			return false, nil
		}
		return false, fmt.Errorf("failed to check installation method: %w", err)
	}

	method := cm.Data[k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey]
	if method == string(installationmethod.K8sInstallationMethodOdigosCli) {
		return true, nil
	}

	return false, nil
}
