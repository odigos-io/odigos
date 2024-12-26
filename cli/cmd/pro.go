package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		tokenType := "onprem" 
		tokenValue := onPremToken

		err = updateOdigosToken(ctx, client, ns, tokenType, tokenValue)
		if err != nil {
			fmt.Println("\033[31mERROR\033[0m Failed to update token:")
			fmt.Println(err)
			os.Exit(1)
		}
		
		fmt.Println()
		fmt.Printf("\u001B[32mSUCCESS:\u001B[0m Token updated successfully in namespace %s\n", ns)
		fmt.Println()
		fmt.Println("The new token will take effect only after the Odiglets are restarted.")
		fmt.Println("To trigger a restart, run the following command:")
		fmt.Println("kubectl rollout restart daemonset odiglet -n", ns)
	},
}

func updateOdigosToken(ctx context.Context, client *kube.Client, namespace string, tokenType, tokenValue string) error {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, consts.OdigosProSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsForbidden(err) {
			return fmt.Errorf("Insufficient permissions to retrieve the token")
		}
		return fmt.Errorf("Tokens are not available in the open-source version of Odigos. Please use the on-premises version.")
	}
	secret.Data[consts.OdigosOnpremTokenSecretKey] = []byte(tokenValue)

	_, err = client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		if apierrors.IsForbidden(err) {
			return fmt.Errorf("Insufficient permissions to update the token")
		}
		return fmt.Errorf("%v", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(proCmd)

	// Flags for update-token
	proCmd.Flags().String("onprem-token", "", "On-prem token for Odigos")
	proCmd.MarkFlagRequired("onprem-token")
}
