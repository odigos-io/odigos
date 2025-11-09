package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/helm"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

var helmInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and upgrade Odigos",
	Long: `This subcommand installs or upgrades Odigos in your Kubernetes cluster.
It installs Kubernetes components that auto-instrument your applications with OpenTelemetry
and send traces, metrics, and logs to any telemetry backend.`,
	RunE: runInstallOrUpgradeWithLegacyCheck,
	Example: `
# Install or upgrade Odigos open-source in your cluster with the default values
odigos install

# Install or upgrade Odigos on-prem tier for enterprise users
odigos install --set onPremToken=${ODIGOS_TOKEN}

# Install or upgrade Odigos and set specific values
odigos install --set collectorGateway.minReplicas=5 --set collectorGateway.maxReplicas=10

# Install or upgrade Odigos using a local values.yaml file
odigos install --values ./values.yaml

# Reset all values to chart defaults (opt out of reuse)
odigos install --reset-then-reuse-values=false
`,
}

var helmUpgradeCmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"update"},
	Short:   "Upgrade Odigos",
	Long: `Upgrades (or installs) Odigos in your Kubernetes cluster.
This command behaves identically to 'odigos install' and uses the same Helm-based flow.`,
	RunE: runInstallOrUpgradeWithLegacyCheck,
	Example: `
# Upgrade Odigos
odigos upgrade

# Upgrade Odigos with custom values
odigos upgrade --set collectorGateway.maxReplicas=10

# Reset all values to chart defaults (opt out of reuse)
odigos upgrade --reset-then-reuse-values=false
`,
}

func runInstallOrUpgradeWithLegacyCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	kubeClient := cmdcontext.KubeClientFromContextOrExit(ctx)

	isLegacy, err := helm.IsLegacyInstallation(ctx, kubeClient.Clientset.CoreV1(), helm.HelmNamespace)
	if err != nil {
		return err
	}
	if isLegacy {
		msg := fmt.Sprintf(`
⚠️  Detected that Odigos was originally installed using an older CLI-based method (without Helm) in namespace "%s".

The current version of the Odigos CLI installs and upgrades Odigos using Helm under the hood,
and cannot automatically upgrade installations created with the legacy method.

👉  To proceed, please do one of the following:
   • Run 'odigos uninstall-deprecated' to remove the old installation, then reinstall using 'odigos install'
   • Or continue using 'odigos upgrade-deprecated' until you are ready to migrate

`, helm.HelmNamespace)

		fmt.Printf("%s\n", msg)
		os.Exit(1)
		return nil
	}

	return runInstallOrUpgrade()
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

	get := action.NewGet(actionConfig)
	_, getErr := get.Run(helm.HelmReleaseName)
	if getErr != nil {
		if errors.Is(getErr, driver.ErrReleaseNotFound) {
			// Release does not exist → install
			rel, err := runInstall(actionConfig, ch, vals)
			if err == nil {
				fmt.Printf("\n✅ Installed release %q in namespace %q (chart version: %s)\n",
					rel.Name, helm.HelmNamespace, ch.Metadata.Version)
			}
			return err
		}
		return getErr // Some other error
	}

	// Release exists → upgrade
	rel, err := runUpgrade(actionConfig, ch, vals)
	if err != nil {
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
	// This ensures defaults are reset but user-provided values are reused.
	upgrade.ResetThenReuseValues = helm.HelmResetThenReuseValues

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
	rootCmd.AddCommand(helmUpgradeCmd)

	for _, c := range []*cobra.Command{helmInstallCmd, helmUpgradeCmd} {
		c.Flags().StringVar(&helm.HelmReleaseName, "release-name", "odigos", "Helm release name")
		c.Flags().StringVarP(&helm.HelmNamespace, "ns", "", "odigos-system", "Target Kubernetes namespace")
		c.Flags().StringVar(&helm.HelmChart, "chart", k8sconsts.DefaultHelmChart, "Helm chart to install (repo/name, local path, or URL)")
		c.Flags().StringVarP(&helm.HelmValuesFile, "values", "f", "", "Path to a custom values.yaml file")
		c.Flags().StringSliceVar(&helm.HelmSetArgs, "set", []string{}, "Set values on the command line (key=value)")
		c.Flags().StringVar(&helm.HelmChartVersion, "chart-version", "", "Override Helm chart version (defaults to CLI-baked version)")
		c.Flags().BoolVar(
			&helm.HelmResetThenReuseValues,
			"reset-then-reuse-values",
			true,
			"Reset to chart defaults, then reuse values from the previous release (default: true).",
		)
	}
}
