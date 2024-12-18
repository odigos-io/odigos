package cmd

import (
	"os"

	"github.com/odigos-io/odigos/cli/pkg/autodetect"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "odigos",
	Short: "Automate OpenTelemetry Observability in Kubernetes",
	Long: `Odigos is primarily focused on automating OpenTelemetry observability pipelines for traces, metrics, and logs, without the need for extensive code changes. The core of Odigos functionality lies in the Kubernetes operators it deploys within your cluster, enabling seamless observability.

Key Features of Odigos:
- Automatic creation of OpenTelemetry observability pipelines.
- Simplified tracing, metrics, and log collection.
- Enhanced visibility into your Kubernetes services.
- Streamlined Kubernetes operations with observability at the forefront.

Get started with Odigos today to effortlessly improve the observability of your Kubernetes services!`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		client := kube.GetCLIClientOrExit(cmd)
		ctx = cmdcontext.ContextWithKubeClient(ctx, client)

		details := autodetect.GetK8SClusterDetails(ctx, kubeConfig, kubeContext, client)
		ctx = cmdcontext.ContextWithClusterDetails(ctx, details)

		cmd.SetContext(ctx)
	},
}

var (
	kubeConfig  string
	kubeContext string
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", env.GetDefaultKubeConfigPath(), "(optional) absolute path to the kubeconfig file")
	rootCmd.PersistentFlags().StringVar(&kubeContext, "kube-context", "", "(optional) name of the kubeconfig context to use")
}
