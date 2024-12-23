package cmd

import (
	"context"
	"fmt"
	"os"

	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// updateTokenCmd represents the update-token command
var updateTokenCmd = &cobra.Command{
	Use:   "update-token",
	Short: "Update a client's token for Odigos",
	Long: `Update the token used for authenticating with Odigos.
This command is useful for updating your on-prem or cloud token for an existing installation.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		// Retrieve namespace
		ns := cmd.Flag("namespace").Value.String()
		if ns == "" {
			fmt.Println("\033[31mERROR\033[0m Namespace must be provided")
			os.Exit(1)
		}

		// Retrieve token
		onPremToken := cmd.Flag("onprem-token").Value.String()
		cloudApiKey := cmd.Flag("cloud-api-key").Value.String()

		if onPremToken == "" && cloudApiKey == "" {
			fmt.Println("\033[31mERROR\033[0m Either --onprem-token or --cloud-api-key must be provided")
			os.Exit(1)
		}
		// todo: check if i should support both onprem and cloud tokens

		// Validate token type
		// var tokenType, tokenValue string
		// if cloudApiKey != "" {
		// 	tokenType = "cloud"
		// 	tokenValue = cloudApiKey
		// 	err := verifyOdigosCloudApiKey(cloudApiKey)
		// 	if err != nil {
		// 		fmt.Println("Odigos update-token failed - invalid cloud API key format.")
		// 		os.Exit(1)
		// 	}
		// } else {
		// 	tokenType = "onprem"
		// 	tokenValue = onPremToken
		// }

		//todo: check what is the client and how to use it
		// Connect to Kubernetes client
		client := cmdcontext.KubeClientFromContextOrExit(ctx)

		// Check if Odigos is installed in the namespace
		_, err := client.CoreV1().ConfigMaps(ns).Get(ctx, k8sconsts.OdigosDeploymentConfigMapName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Odigos is not installed in namespace %s\n", ns)
			os.Exit(1)
		}

		// Update token
		//		err = updateOdigosToken(ctx, client, ns, tokenType, tokenValue)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to update token: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("\u001B[32mSUCCESS:\u001B[0m Token updated successfully in namespace %s\n", ns)
	},
}

// todo: should i assume that the secret is in the odigos-system namespace?
// updateOdigosToken updates the "odigos-pro" secret in the "odigos-system" namespace
func updateOdigosToken(ctx context.Context, client kubernetes.Interface, tokenType, tokenValue string) error {
	secretName := "odigos-pro"
	namespace := "odigos-system"

	// Retrieve the existing secret
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %s in namespace %s: %w", secretName, namespace, err)
	}

	// Update the secret with the new token
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data["token"] = []byte(tokenValue)
	secret.Data["type"] = []byte(tokenType)

	// Apply the updated secret
	_, err = client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret %s in namespace %s: %w", secretName, namespace, err)
	}

	fmt.Printf("Updated secret %s in namespace %s with new %s token\n", secretName, namespace, tokenType)
	return nil
}

func init() {
	rootCmd.AddCommand(updateTokenCmd)

	// Flags for update-token
	updateTokenCmd.Flags().StringP("namespace", "n", "default", "Kubernetes namespace where Odigos is installed")
	updateTokenCmd.Flags().String("onprem-token", "", "On-prem token for Odigos")
	updateTokenCmd.Flags().String("cloud-api-key", "", "Cloud API key for Odigos")
}
