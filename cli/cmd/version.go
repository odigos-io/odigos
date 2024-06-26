/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cliFlag     = "cli"
	clusterFlag = "cluster"
)

var (
	OdigosVersion        string
	OdigosClusterVersion string
	OdigosCommit         string
	OdigosDate           string
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print odigos version.",
	Run: func(cmd *cobra.Command, args []string) {
		short_flag, _ := cmd.Flags().GetBool(cliFlag)

		if short_flag {
			fmt.Printf("%s\n", OdigosVersion)
			return
		}

		client, ns, err := getOdigosKubeClientAndNamespace(cmd)
		if err == nil {
			OdigosClusterVersion, _ = GetOdigosVersionInCluster(cmd.Context(), client, ns)
		}

		cluster_flag, _ := cmd.Flags().GetBool(clusterFlag)

		if cluster_flag {
			fmt.Printf("%s\n", OdigosClusterVersion)
			return
		}

		fmt.Printf("Odigos Cli Version: version.Info{Version:'%s', GitCommit:'%s', BuildDate:'%s'}\n", OdigosVersion, OdigosCommit, OdigosDate)
		fmt.Printf("Odigos Version (in cluster): version.Info{Version:'%s'}\n", OdigosClusterVersion)
	},
}

func GetOdigosVersionInCluster(ctx context.Context, client *kube.Client, ns string) (string, error) {
	cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, resources.OdigosDeploymentConfigMapName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error detecting Odigos version in the current cluster")
	}

	odigosVersion, ok := cm.Data["ODIGOS_VERSION"]
	if !ok || odigosVersion == "" {
		return "", fmt.Errorf("error detecting Odigos version in the current cluster")
	}

	return odigosVersion, nil
}

func getOdigosKubeClientAndNamespace(cmd *cobra.Command) (*kube.Client, string, error) {
	client, err := kube.CreateClient(cmd)
	if err != nil {
		return nil, "", err
	}
	ctx := cmd.Context()

	ns, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		if resources.IsErrNoOdigosNamespaceFound(err) {
			fmt.Println("Odigos is NOT yet installed in the current cluster")
		} else {
			fmt.Println("Error detecting Odigos version in the current cluster")
		}
		return nil, "", err
	}

	return client, ns, nil
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().Bool(cliFlag, false, "prints only the CLI version")
	versionCmd.Flags().Bool(clusterFlag, false, "prints only the Cluster version")
}
