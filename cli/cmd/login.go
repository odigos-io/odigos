package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/labels"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/odigos-io/odigos/common"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// This function requires knowledge of which objects should be updated once the odigos cloud secret is updated.
// It is not very generic, but it is the best we can do for now.
func restartPodsAfterCloudLogin(ctx context.Context, client *kube.Client, ns string, configVersion int) error {

	configVersionStr := strconv.Itoa(configVersion)
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"%s":"%s"}}}}}`, labels.OdigosSystemConfigLabelKey, configVersionStr)

	_, err := client.AppsV1().Deployments(ns).Patch(ctx, resources.KeyvalProxyDeploymentName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return err
	}

	_, err = client.AppsV1().Deployments(ns).Patch(ctx, resources.OwnTelemetryCollectorDeploymentName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return err
	}

	_, err = client.AppsV1().DaemonSets(ns).Patch(ctx, resources.OdigletDaemonSetName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

// both login and update trigger this function.
func updateApiKey(cmd *cobra.Command, args []string) {
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

	config, err := resources.GetCurrentConfig(ctx, client, ns)
	if err != nil {
		fmt.Println("Odigos cloud login failed - unable to read the current Odigos configuration.")
		os.Exit(1)
	}
	config.Spec.ConfigVersion += 1

	if odigosCloudApiKeyFlag == "" {
		fmt.Println("Enter your odigos cloud api-key. You can find it here: https://app.odigos.io/settings")
		fmt.Print("api-key: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		odigosCloudApiKeyFlag = scanner.Text()
	}

	err = verifyOdigosCloudApiKey(odigosCloudApiKeyFlag)
	if err != nil {
		fmt.Println("Odigos cloud login failed - invalid api-key format.")
		os.Exit(1)
	}

	currentTier, err := odigospro.GetCurrentOdigosTier(ctx, client, ns)
	if err != nil {
		fmt.Println("Odigos cloud login failed - unable to read the current Odigos tier.")
		os.Exit(1)
	}
	if currentTier == common.OnPremOdigosTier {
		fmt.Println("You are using on premises version of Odigos. Contact your Odigos representative to enable odigos cloud.")
		return
	}
	isPrevOdigosCloud := currentTier == common.CloudOdigosTier

	resourceManagers := resources.CreateResourceManagers(client, ns, common.CloudOdigosTier, &odigosCloudApiKeyFlag, &config.Spec)
	err = resources.ApplyResourceManagers(ctx, client, resourceManagers, "Updating")
	if err != nil {
		fmt.Println("Odigos cloud login failed - unable to apply Odigos resources.")
		os.Exit(1)
	}
	err = resources.DeleteOldOdigosSystemObjects(ctx, client, ns, config)
	if err != nil {
		fmt.Println("Odigos cloud login failed - unable to cleanup old Odigos resources.")
		os.Exit(1)
	}

	if isPrevOdigosCloud {
		l := log.Print("Restarting relevant pods ...")
		err := restartPodsAfterCloudLogin(ctx, client, ns, config.Spec.ConfigVersion)
		if err != nil {
			l.Error(err)
		}
		l.Success()
	}
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Odigos cloud",
	Long:  `Connect this Odigos installation to your odigos cloud account.`,
	Run:   updateApiKey,
}

// loginCmd represents the login command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update your Odigos Cloud api-key",
	Long:  `Use this command to update your Odigos Cloud api-key.`,
	Run:   updateApiKey,
}

func init() {
	cloudCmd.AddCommand(loginCmd)
	cloudCmd.AddCommand(updateCmd)

	loginCmd.Flags().StringVarP(&odigosCloudApiKeyFlag, "api-key", "k", "", "api key for odigos cloud")
	updateCmd.Flags().StringVarP(&odigosCloudApiKeyFlag, "api-key", "k", "", "api key for odigos cloud")
}
