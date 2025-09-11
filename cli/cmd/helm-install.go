package cmd

import (
	"fmt"

	"github.com/odigos-io/odigos/cli/pkg/helm"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

var helmInstallCmd = &cobra.Command{
	Use:   "helm-install",
	Short: "Install Odigos using Helm under the hood",
	Long:  `Installs Odigos in your cluster. Equivalent to "helm install" with some defaults.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHelmInstall()
	},
}

func runHelmInstall() error {
	settings := cli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), helm.HelmNamespace, "secret", helm.DebugLogger); err != nil {
		return err
	}

	ch, vals, err := helm.PrepareChartAndValues(settings)
	if err != nil {
		return err
	}

	install := action.NewInstall(actionConfig)
	install.ReleaseName = helm.HelmReleaseName
	install.Namespace = helm.HelmNamespace
	install.CreateNamespace = true
	install.ChartPathOptions.Version = ch.Metadata.Version

	rel, err := install.Run(ch, vals)
	if err != nil {
		return err
	}

	fmt.Printf("\n✅ Installed release %q in namespace %q (chart version: %s)\n",
		rel.Name, helm.HelmNamespace, ch.Metadata.Version)
	return nil
}

func init() {
	rootCmd.AddCommand(helmInstallCmd)

	helmInstallCmd.Flags().StringVar(&helm.HelmReleaseName, "release-name", "odigos", "Helm release name")
	helmInstallCmd.Flags().StringVarP(&helm.HelmNamespace, "namespace", "n", "odigos-system", "Target Kubernetes namespace")
	helmInstallCmd.Flags().StringVar(&helm.HelmChart, "chart", "odigos/odigos", "Helm chart to install (repo/name, local path, or URL)")
	helmInstallCmd.Flags().StringVarP(&helm.HelmValuesFile, "values", "f", "", "Path to a custom values.yaml file")
	helmInstallCmd.Flags().StringSliceVar(&helm.HelmSetArgs, "set", []string{}, "Set values on the command line (key=value)")
	helmInstallCmd.Flags().StringVar(&helm.HelmChartVersion, "chart-version", "", "Override Helm chart version (defaults to CLI-baked version)")
}
