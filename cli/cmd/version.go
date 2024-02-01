/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	OdigosVersion string
	OdigosCommit  string
	OdigosDate    string
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print odigos version.",
	Run: func(cmd *cobra.Command, args []string) {
		short_flag, _ := cmd.Flags().GetBool("short")

		if short_flag {
			fmt.Printf("%s\n", OdigosVersion)
			return
		}

		fmt.Printf("Odigos Cli Version: version.Info{Version:'%s', GitCommit:'%s', BuildDate:'%s'}\n", OdigosVersion, OdigosCommit, OdigosDate)
		printOdigosClusterVersion(cmd)
	},
}

func printOdigosClusterVersion(cmd *cobra.Command) {
	client, err := kube.CreateClient(cmd)
	if err != nil {
		return
	}
	ctx := cmd.Context()

	ns, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("Odigos is NOT yet installed in the current cluster")
		} else {
			fmt.Println("Error detecting Odigos version in the current cluster")
		}
		return
	}

	cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, resources.OdigosDeploymentConfigMapName, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Error detecting Odigos version in the current cluster")
		return
	}

	odigosVersion, ok := cm.Data["ODIGOS_VERSION"]
	if !ok || odigosVersion == "" {
		fmt.Println("Error detecting Odigos version in the current cluster")
		return
	}
	fmt.Printf("Odigos Version (in cluster): version.Info{Version:'%s'}\n", odigosVersion)
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.PersistentFlags().Bool("short", false, "prints only the CLI version")
}
