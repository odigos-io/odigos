package cmd

import (
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	KUBECONFIG = "KUBECONFIG"
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
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

var (
	kubeConfig string
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func getDefaultKubeConfigPath() string {
	if val, ok := os.LookupEnv(KUBECONFIG); ok {
		return val
	} else {
		if home := homedir.HomeDir(); home != "" {
			return filepath.Join(home, ".kube", "config")
		}
	}
	return ""
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", getDefaultKubeConfigPath(), "(optional) absolute path to the kubeconfig file")
}
