package cmd

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/cli/pkg/helm"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
)

var helmInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and upgrade Odigos using Helm SDK under the hood",
	Long:  `This sub command will Install and upgrade Odigos in your kubernetes cluster. It will install k8s components that will auto-instrument your applications with OpenTelemetry and send traces, metrics and logs to any telemetry backend.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInstallOrUpgrade()
	},
	Example: `
# Install or upgrade Odigos open-source in your cluster with the default values.
odigos install

# Install or upgrade Odigos onprem tier for enterprise users
odigos install --set onPremToken=${ODIGOS_TOKEN}

# Install or upgrade Odigos and set specific values.
odigos install --set collectorGateway.minReplicas=5 --set collectorGateway.maxReplicas=10

# Install or upgrade Odigos and use local values.yaml file.
odigos install --values ./values.yaml
`,
}

func runInstallOrUpgrade() error {
	settings := cli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), helm.HelmNamespace, "secret", helm.CustomInstallLogger); err != nil {
		return err
	}

	ch, vals, err := helm.PrepareChartAndValues(settings)
	if err != nil {
		return err
	}

	// Helm SDK note:
	// Unlike the `helm upgrade --install` CLI command, the Go SDK's Upgrade action
	// does NOT support automatically creating a release when it doesn't exist.
	//
	// If the release is missing, the SDK returns an error message string such as:
	//   "release: not found"
	//   "has no deployed releases"
	//
	// Because the SDK does not provide a typed error, the only way to detect
	// this case is to check the error message text and explicitly fall back
	// to running an Install action instead.
	//
	// if Helm changes its error messages in the future, we may need to update these checks.
	rel, err := runUpgrade(actionConfig, ch, vals)
	if err != nil {
		// Fallback if release does not exist
		if strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "no deployed") {
			rel, err = runInstall(actionConfig, ch, vals)
			if err != nil {
				return err
			}
			fmt.Printf("\n✅ Installed release %q in namespace %q (chart version: %s)\n",
				rel.Name, helm.HelmNamespace, ch.Metadata.Version)
			return nil
		}
		return err
	}

	helm.PrintSummary()

	fmt.Printf("\n✅ Upgraded release %q in namespace %q (chart version: %s)\n",
		rel.Name, helm.HelmNamespace, ch.Metadata.Version)
	return nil
}

func runUpgrade(actionConfig *action.Configuration, ch *chart.Chart, vals map[string]interface{}) (*release.Release, error) {
	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = helm.HelmNamespace
	upgrade.Install = false // we handle install fallback ourselves
	upgrade.ChartPathOptions.Version = ch.Metadata.Version
	return upgrade.Run(helm.HelmReleaseName, ch, vals)
}

func runInstall(actionConfig *action.Configuration, ch *chart.Chart, vals map[string]interface{}) (*release.Release, error) {
	install := action.NewInstall(actionConfig)
	install.ReleaseName = helm.HelmReleaseName
	install.Namespace = helm.HelmNamespace
	install.CreateNamespace = true
	install.ChartPathOptions.Version = ch.Metadata.Version
	return install.Run(ch, vals)
}

func init() {
	rootCmd.AddCommand(helmInstallCmd)

	helmInstallCmd.Flags().StringVar(&helm.HelmReleaseName, "release-name", "odigos", "Helm release name")
	helmInstallCmd.Flags().StringVarP(&helm.HelmNamespace, "ns", "", "odigos-system", "Target Kubernetes namespace")
	helmInstallCmd.Flags().StringVar(&helm.HelmChart, "chart", "odigos/odigos", "Helm chart to install (repo/name, local path, or URL)")
	helmInstallCmd.Flags().StringVarP(&helm.HelmValuesFile, "values", "f", "", "Path to a custom values.yaml file")
	helmInstallCmd.Flags().StringSliceVar(&helm.HelmSetArgs, "set", []string{}, "Set values on the command line (key=value)")
	helmInstallCmd.Flags().StringVar(&helm.HelmChartVersion, "chart-version", "", "Override Helm chart version (defaults to CLI-baked version)")
}
