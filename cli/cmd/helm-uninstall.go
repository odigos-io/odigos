package cmd

import (
	"fmt"

	"github.com/odigos-io/odigos/cli/pkg/helm"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

var helmUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Odigos",
	Long:  `Revert all the changes made by the odigos install command. This command will uninstall Odigos from your cluster. It will delete all Odigos objects and rollback any metadata changes made to your objects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHelmUninstall()
	},
	Example: `
# Uninstall Odigos
odigos uninstall
`,
}

func runHelmUninstall() error {
	fmt.Printf("🗑️  Starting uninstall of release %q from namespace %q...\n", helm.HelmReleaseName, helm.HelmNamespace)

	settings := cli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), helm.HelmNamespace, "secret", helm.CustomUninstallLogger); err != nil {
		return err
	}

	res, err := helm.RunUninstall(actionConfig, helm.HelmReleaseName)
	if err != nil {
		return err
	}

	if res == nil {
		// Release was not found, already uninstalled
		fmt.Printf("\n🗑️  Release %q not found in namespace %q (already uninstalled)\n", helm.HelmReleaseName, helm.HelmNamespace)
		return nil
	}

	helm.PrintSummary()

	fmt.Printf("\n🗑️  Uninstalled release %q from namespace %q\n", helm.HelmReleaseName, helm.HelmNamespace)
	if res.Info != "" {
		fmt.Printf("Info: %s\n", res.Info)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(helmUninstallCmd)

	helmUninstallCmd.Flags().StringVar(&helm.HelmReleaseName, "release-name", "odigos", "Helm release name")
	helmUninstallCmd.Flags().StringVarP(&helm.HelmNamespace, "ns", "", "odigos-system", "Target Kubernetes namespace")
}
