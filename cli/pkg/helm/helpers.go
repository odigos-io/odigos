package helm

import (
	"fmt"
	"os"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
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

// prepareChartAndValues loads chart and merges values
func PrepareChartAndValues(settings *cli.EnvSettings) (*chart.Chart, map[string]interface{}, error) {
	// choose version
	version := ""
	if HelmChartVersion != "" {
		version = strings.TrimPrefix(HelmChartVersion, "v")
	} else if OdigosChartVersion != "" {
		version = strings.TrimPrefix(OdigosChartVersion, "v")
	}

	// ensure odigos repo exists if using odigos/ chart
	if strings.HasPrefix(HelmChart, "odigos/") {
		if err := ensureHelmRepo(settings, "odigos", "https://odigos-io.github.io/odigos/"); err != nil {
			return nil, nil, err
		}
	}

	// refresh repo index if using a remote chart
	if strings.Contains(HelmChart, "/") {
		refreshHelmRepo(settings, HelmChart)
	}

	// load chart
	chartPathOptions := action.ChartPathOptions{Version: version}
	chartPath, err := chartPathOptions.LocateChart(HelmChart, settings)
	if err != nil {
		return nil, nil, err
	}
	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, nil, err
	}

	// merge values
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
	if ch.Metadata.AppVersion != "" {
		if _, ok := vals["image"]; !ok {
			vals["image"] = map[string]interface{}{}
		}
		if imgVals, ok := vals["image"].(map[string]interface{}); ok {
			if _, hasTag := imgVals["tag"]; !hasTag || imgVals["tag"] == "" {
				imgVals["tag"] = ch.Metadata.AppVersion
			}
		}
	}

	return ch, vals, nil
}

// ensureHelmRepo adds a repo if missing
func ensureHelmRepo(settings *cli.EnvSettings, name, url string) error {
	repoFile := settings.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// check if exists
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
