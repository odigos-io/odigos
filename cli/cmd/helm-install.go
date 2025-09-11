package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

var (
	helmReleaseName  string
	helmNamespace    string
	helmChart        string
	helmValuesFile   string
	helmSetArgs      []string
	helmChartVersion string
)

// injected at build time with ldflags from .ko.yaml
var OdigosChartVersion string

// helmInstallCmd represents the helm-install command
var helmInstallCmd = &cobra.Command{
	Use:   "helm-install",
	Short: "Install or upgrade Odigos using Helm under the hood",
	Long: `This command installs Odigos into your Kubernetes cluster using Helm.
It wraps "helm upgrade --install" and supports --values and --set just like Helm.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHelmUpgradeInstall()
	},
}

func runHelmUpgradeInstall() error {
	settings := cli.New()
	actionConfig := new(action.Configuration)

	debug := func(format string, v ...interface{}) {
		fmt.Printf(format+"\n", v...)
	}

	if err := actionConfig.Init(settings.RESTClientGetter(), helmNamespace, "secret", debug); err != nil {
		return err
	}

	// prepare chart & values
	ch, vals, err := prepareChartAndValues(settings)
	if err != nil {
		return err
	}

	// run upgrade, fallback to install if needed
	rel, err := runUpgrade(actionConfig, ch, vals)
	if err != nil {
		// fallback conditions from Helm SDK
		if strings.Contains(err.Error(), "release: not found") ||
			strings.Contains(err.Error(), "has no deployed releases") {
			rel, err = runInstall(actionConfig, ch, vals)
			if err != nil {
				return err
			}
			fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Release %q installed in namespace %q (chart version: %s)\n",
				rel.Name, helmNamespace, ch.Metadata.Version)
			return nil
		}
		return err
	}

	fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Release %q upgraded in namespace %q (chart version: %s)\n",
		rel.Name, helmNamespace, ch.Metadata.Version)
	return nil
}

func prepareChartAndValues(settings *cli.EnvSettings) (*chart.Chart, map[string]interface{}, error) {
	// choose version
	version := ""
	if helmChartVersion != "" {
		version = strings.TrimPrefix(helmChartVersion, "v")
	} else if OdigosChartVersion != "" {
		version = strings.TrimPrefix(OdigosChartVersion, "v")
	}

	// ensure odigos repo exists if using odigos/ chart
	if strings.HasPrefix(helmChart, "odigos/") {
		if err := ensureHelmRepo(settings, "odigos", "https://odigos-io.github.io/odigos/"); err != nil {
			return nil, nil, err
		}
	}

	// refresh repo index if using a remote chart
	if strings.Contains(helmChart, "/") {
		refreshHelmRepo(settings, helmChart)
	}

	// load chart
	chartPathOptions := action.ChartPathOptions{Version: version}
	chartPath, err := chartPathOptions.LocateChart(helmChart, settings)
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
		Values:     helmSetArgs,
	}
	if helmValuesFile != "" {
		valOpts.ValueFiles = append(valOpts.ValueFiles, helmValuesFile)
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
				fmt.Printf("Using appVersion %s as image.tag\n", ch.Metadata.AppVersion)
			}
		}
	}

	return ch, vals, nil
}

func runUpgrade(actionConfig *action.Configuration, ch *chart.Chart, vals map[string]interface{}) (*release.Release, error) {
	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = helmNamespace
	upgrade.Install = true
	upgrade.ChartPathOptions.Version = ch.Metadata.Version
	return upgrade.Run(helmReleaseName, ch, vals)
}

func runInstall(actionConfig *action.Configuration, ch *chart.Chart, vals map[string]interface{}) (*release.Release, error) {
	install := action.NewInstall(actionConfig)
	install.ReleaseName = helmReleaseName
	install.Namespace = helmNamespace
	install.CreateNamespace = true
	install.ChartPathOptions.Version = ch.Metadata.Version
	return install.Run(ch, vals)
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

func init() {
	rootCmd.AddCommand(helmInstallCmd)

	helmInstallCmd.Flags().StringVar(&helmReleaseName, "release-name", "odigos", "Helm release name")
	helmInstallCmd.Flags().StringVarP(&helmNamespace, "namespace", "n", "odigos-system", "Target Kubernetes namespace")
	helmInstallCmd.Flags().StringVar(&helmChart, "chart", "odigos/odigos", "Helm chart to install (repo/name, local path, or URL)")
	helmInstallCmd.Flags().StringVarP(&helmValuesFile, "values", "f", "", "Path to a custom values.yaml file")
	helmInstallCmd.Flags().StringSliceVar(&helmSetArgs, "set", []string{}, "Set values on the command line (key=value)")
	helmInstallCmd.Flags().StringVar(&helmChartVersion, "chart-version", "", "Override Helm chart version (defaults to CLI-baked version)")
}
