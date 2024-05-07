package cmd

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/spf13/cobra"
)

func verifyOdigosCloudApiKey(apikey string) error {
	_, err := uuid.Parse(apikey)
	if err != nil {
		return fmt.Errorf("invalid apikey format. expected uuid format")
	}

	return nil
}

// cloudCmd represents the cloud command
var cloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Manage odigos cloud",
	Long:  `Used to interact with odigos managed service.`,
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
			fmt.Println("Error reading current odigos tier")
			os.Exit(1)
		}

		if currentTier == common.CloudOdigosTier {
			fmt.Println("Odigos cloud is currently enabled")
		} else if currentTier == common.CommunityOdigosTier {
			fmt.Println(`You are using the community version of Odigos.
To enable odigos cloud run 'odigos cloud login'`)
		} else {
			fmt.Println(`You are using the on-prem enterprise version of Odigos.
Contact your Odigos representative to enable odigos cloud`)
		}
	},
}

func init() {
	rootCmd.AddCommand(cloudCmd)
}
