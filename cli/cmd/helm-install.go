package cmd

import (
	"fmt"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/helm"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
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

	ch, vals, err := helm.PrepareChartAndValues(settings, k8sconsts.OdigosHelmRepoName)
	if err != nil {
		return err
	}

	result, err := helm.InstallOrUpgrade(actionConfig, ch, vals, helm.HelmNamespace, helm.HelmReleaseName, helm.InstallOrUpgradeOptions{
		CreateNamespace:      true,
		ResetThenReuseValues: helm.HelmResetThenReuseValues,
	})
	if err != nil {
		return err
	}

	helm.PrintSummary()

	fmt.Printf("\n✅ %s\n", helm.FormatInstallOrUpgradeMessage(result, ch.Metadata.Version))
	return nil
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
