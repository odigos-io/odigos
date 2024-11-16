/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/getters"
	"github.com/spf13/cobra"
)

const (
	cliFlag     = "cli"
	clusterFlag = "cluster"
)

var (
	OdigosVersion        string
	odigosClusterVersion string
	OdigosCommit         string
	OdigosDate           string
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print odigos version.",
	Run: func(cmd *cobra.Command, args []string) {
		cliFlag, _ := cmd.Flags().GetBool(cliFlag)
		clusterFlag, _ := cmd.Flags().GetBool(clusterFlag)

		if cliFlag {
			fmt.Printf("%s\n", OdigosVersion)
		}

		OdigosClusterVersion, err := getOdigosVersionInCluster(cmd)

		if clusterFlag && err == nil {
			fmt.Printf("%s\n", OdigosClusterVersion)
		}

		if cliFlag || clusterFlag {
			return
		}

		if err != nil {
			fmt.Printf("%s\n", err)
		}

		fmt.Printf("Odigos Cli Version: version.Info{Version:'%s', GitCommit:'%s', BuildDate:'%s'}\n", OdigosVersion, OdigosCommit, OdigosDate)
		fmt.Printf("Odigos Version (in cluster): version.Info{Version:'%s'}\n", OdigosClusterVersion)

	},
}

func getOdigosVersionInCluster(cmd *cobra.Command) (string, error) {
	client, ns, err := getOdigosKubeClientAndNamespace(cmd)
	if err != nil {
		return "", err
	}

	return getters.GetOdigosVersionInClusterFromConfigMap(cmd.Context(), client.Clientset, ns)
}

func getOdigosKubeClientAndNamespace(cmd *cobra.Command) (*kube.Client, string, error) {
	ctx := cmd.Context()
	client := kube.KubeClientFromContextOrExit(ctx)

	ns, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		if resources.IsErrNoOdigosNamespaceFound(err) {
			err = fmt.Errorf("Odigos is NOT yet installed in the current cluster")
		} else {
			err = fmt.Errorf("Error detecting Odigos namespace in the current cluster")
		}
	}

	return client, ns, err
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().Bool(cliFlag, false, "prints only the CLI version")
	versionCmd.Flags().Bool(clusterFlag, false, "prints only the version of odigos deployed in the cluster")
}
