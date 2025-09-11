package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
)

var (
	helmReleaseName  string
	helmNamespace    string
	helmChart        string
	helmValuesFile   string
	helmSetArgs      []string
	helmChartVersion string
)

// these will be injected at build time with ldflags from .ko.yaml
var (
	OdigosChartVersion string
)

// helmInstallCmd represents the helm-install command
var helmInstallCmd = &cobra.Command{
	Use:   "helm-install",
	Short: "Install or upgrade Odigos using Helm under the hood",
	Long: `This command installs Odigos into your Kubernetes cluster using Helm.
It wraps "helm upgrade --install" and supports --values and --set just like Helm.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHelmUpgradeInstall()
	},
	Example: `
# Install Odigos using the baked-in chart version
odigos helm-install --release-name odigos --namespace odigos-system --chart odigos/odigos

# Install Odigos with a custom values.yaml
odigos helm-install --release-name odigos --namespace odigos-system --chart odigos/odigos --values my-values.yaml

# Install Odigos and override specific values
odigos helm-install --release-name odigos --namespace odigos-system --chart odigos/odigos --set replicaCount=3,image.tag=latest
`,
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

	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = helmNamespace
	upgrade.Install = true // enables upgrade --install

	// Pin to baked-in chart version unless user explicitly sets --chart-version
	if helmChartVersion != "" {
		upgrade.ChartPathOptions.Version = helmChartVersion
	} else if OdigosChartVersion != "" {
		upgrade.ChartPathOptions.Version = OdigosChartVersion
	}

	chartPath, err := upgrade.ChartPathOptions.LocateChart(helmChart, settings)
	if err != nil {
		return err
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	valOpts := &values.Options{
		ValueFiles: []string{},
		Values:     helmSetArgs,
	}
	if helmValuesFile != "" {
		valOpts.ValueFiles = append(valOpts.ValueFiles, helmValuesFile)
	}

	vals, err := valOpts.MergeValues(getter.All(settings))
	if err != nil {
		return err
	}

	rel, err := upgrade.Run(helmReleaseName, chart, vals)
	if err != nil {
		return err
	}

	fmt.Printf("\n\u001B[32mSUCCESS:\u001B[0m Release %q installed/updated in namespace %q (chart version: %s)\n",
		rel.Name, helmNamespace, upgrade.ChartPathOptions.Version)

	return nil
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
