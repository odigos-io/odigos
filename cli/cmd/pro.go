package cmd

import (
	"context"
	"fmt"
	"os"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/spf13/cobra"
)

var (
	updateRemoteFlag    bool
)

var proCmd = &cobra.Command{
	Use:   "pro",
	Short: "manage odigos pro",
	Long:  `The pro command provides various operations and functionalities specifically designed for enterprise users. Use this command to access advanced features and manage your pro account.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		ns, err := resources.GetOdigosNamespace(client, ctx)
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("\033[31mERROR\033[0m no odigos installation found in the current cluster")
			os.Exit(1)
		} else if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to check if Odigos is already installed: %s\n", err)
			os.Exit(1)
		}
		onPremToken := cmd.Flag("onprem-token").Value.String()
		if updateRemoteFlag{
			err = executeRemoteUpdateToken(ctx, client, ns)
		} else {
			err = pro.UpdateOdigosToken(ctx, client, ns, onPremToken)
		}

		if err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to update token:")
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Println()
			fmt.Println("\u001B[32mSUCCESS:\u001B[0m Token updated successfully")
		}
	},
}

func executeRemoteUpdateToken(ctx context.Context, client *kube.Client, namespace string) error {
	uiSvcProxyEndpoint := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/api/pro/source/namespace/%s/kind/%s/name/%s", namespace, k8sconsts.OdigosUiServiceName, k8sconsts.OdigosUiServicePort)
	request := client.Clientset.RESTClient().Get().AbsPath(uiSvcProxyEndpoint).Do(ctx)
	_, err := request.Raw()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func init() {
	rootCmd.AddCommand(proCmd)

	proCmd.Flags().String("onprem-token", "", "On-prem token for Odigos")
	proCmd.MarkFlagRequired("onprem-token")
	proCmd.PersistentFlags().BoolVarP(&updateRemoteFlag, "remote", "r", false, "use odigos ui service in the cluster to update the onprem token")
}
