package cmd

import (
	"fmt"
	"os"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/pkg/confirm"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
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

		currentTier, err := odigospro.GetCurrentOdigosTier(ctx, client, ns)
		if err != nil {
			fmt.Println("Odigos cloud login failed - unable to read the current Odigos tier.")
			os.Exit(1)
		}
		if currentTier == common.CommunityOdigosTier {
			fmt.Println("Odigos is already in community tier.")
			os.Exit(1)
		}
		if currentTier == common.OnPremOdigosTier {
			fmt.Println("Odigos tier is on-premises. Contact your Odigos representative to switch to community tier.")
			os.Exit(1)
		}

		fmt.Println("About to logout from Odigos cloud. You can still manager your Odigos installation locally with 'odigos ui'.")
		if !cmd.Flag("yes").Changed {
			confirmed, err := confirm.Ask("Are you sure?")
			if err != nil || !confirmed {
				fmt.Println("Aborting odigos cloud logout")
				return
			}
		}

		config, err := resources.GetCurrentConfig(ctx, client, ns)
		if err != nil {
			fmt.Println("Odigos cloud logout failed - unable to read the current Odigos configuration.")
			os.Exit(1)
		}
		config.Spec.ConfigVersion += 1

		emptyApiKey := ""
		resourceManagers := resources.CreateResourceManagers(client, ns, common.CommunityOdigosTier, &emptyApiKey, &config.Spec)
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
	logoutCmd.Flags().Bool("yes", false, "skip the confirmation prompt")
}
