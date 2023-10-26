package cmd

import (
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "odigos",
	Short: "A CLI tool for managing your Kubernetes resources",
	Long: `Odigos is a command-line tool for simplifying Kubernetes resource management.
It allows you to interact with your Kubernetes cluster, manage resources, and perform various operations.

You can use Odigos to:
- Create and manage Kubernetes resources.
- Deploy applications to your cluster.
- Monitor the status of your workloads.
- Configure and customize your Kubernetes environment.

Odigos is designed to make Kubernetes operations easy and efficient. Get started with Odigos today!`,
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

func init() {
	if home := homedir.HomeDir(); home != "" {
		rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", "", "(optional) absolute path to the kubeconfig file")
	}
}
