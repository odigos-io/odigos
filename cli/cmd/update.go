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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// updateTokenCmd represents the update-token command
var updateTokenCmd = &cobra.Command{
	Use:   "update-token",
	Short: "Update a client's token for Odigos",
	Long: `Update the token used for authenticating with Odigos.
This command is useful for updating your on-prem or cloud token for an existing installation.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		client := cmdcontext.KubeClientFromContextOrExit(ctx)
		ns, err := resources.GetOdigosNamespace(client, ctx)
		if err != nil {
			fmt.Errorf("No Odigos installation found in cluster to upgrade")
			os.Exit(1)
		}

		// Retrieve token
		onPremToken := cmd.Flag("onprem-token").Value.String()

		if onPremToken == "" {
			fmt.Errorf("\033[31mERROR\033[0m --onprem-token is required")
			os.Exit(1)
		}

		var tokenType, tokenValue string = "onprem", onPremToken
		err = updateOdigosToken(ctx, client, ns, tokenType, tokenValue)
		if err != nil {
			fmt.Errorf("\033[31mERROR\033[0m Failed to update token: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("\u001B[32mSUCCESS:\u001B[0m Token updated successfully in namespace %s\n", ns)
	},
}

func updateOdigosToken(ctx context.Context, client *kube.Client, namespace string, tokenType, tokenValue string) error {

	// Retrieve the existing secret
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, consts.OdigosProSecretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %s in namespace %s: %w", consts.OdigosProSecretName, namespace, err)
	}

	secret.Data["odigos-onprem-token"] = []byte(tokenValue)

	// Apply the updated secret
	_, err = client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret %s in namespace %s: %w", consts.OdigosProSecretName, namespace, err)
	}

	fmt.Printf("Updated secret %s in namespace %s with new %s token\n", consts.OdigosProSecretName, namespace, tokenType)
	return nil
}

func init() {
	rootCmd.AddCommand(updateTokenCmd)

	// Flags for update-token
	updateTokenCmd.Flags().String("onprem-token", "", "On-prem token for Odigos")
}
