package cmd

import (
	"fmt"
	"os"

	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/pkg/confirm"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/spf13/cobra"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Odigos cloud",
	Long: `Disconnect this Odigos installation from your odigos cloud account.
	
	After running this command, you will no longer be able to control and monitor this Odigos installation from Odigos cloud.
	You can run 'odigos ui' to manage your Odigos installation locally.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := kube.CreateClient(cmd)
		if err != nil {
			kube.PrintClientErrorAndExit(err)
		}
		ctx := cmd.Context()

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("\033[31mERROR\033[0m no odigos installation found in the current cluster. use \"odigos install\" to install odigos in the cluster or check that kubeconfig is pointing to the correct cluster.")
			os.Exit(1)
		} else if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already installed: %s\n", err)
			os.Exit(1)
		}

		isOdigosCloud, err := resources.IsOdigosCloud(ctx, client, ns)
		if err != nil {
			fmt.Println("Odigos cloud logout failed - unable to read the current Odigos cloud configuration.")
			os.Exit(1)
		}
		if !isOdigosCloud {
			fmt.Println("The current odigos installation is not connected to Odigos cloud.")
			os.Exit(1)
		}

		fmt.Println("About to logout from Odigos cloud. You can still manager your Odigos installation locally with 'odigos ui'.")
		confirmed, err := confirm.Ask("Are you sure?")
		if err != nil || !confirmed {
			fmt.Println("Aborting odigos cloud logout")
			return
		}

		config, err := resources.GetCurrentConfig(ctx, client, ns)
		if err != nil {
			fmt.Println("Odigos cloud logout failed - unable to read the current Odigos configuration.")
			os.Exit(1)
		}
		config.Spec.ConfigVersion += 1

		emptyApiKey := ""
		resourceManagers := resources.CreateResourceManagers(client, ns, false, &emptyApiKey, &config.Spec)
		err = resources.ApplyResourceManagers(ctx, client, resourceManagers, "Updating")
		if err != nil {
			fmt.Println("Odigos cloud logout failed - unable to apply Odigos resources.")
			os.Exit(1)
		}
		err = resources.DeleteOldOdigosSystemObjects(ctx, client, ns, config)
		if err != nil {
			fmt.Println("Odigos cloud logout failed - unable to cleanup old Odigos resources.")
			os.Exit(1)
		}
	},
}

func init() {
	cloudCmd.AddCommand(logoutCmd)
}
