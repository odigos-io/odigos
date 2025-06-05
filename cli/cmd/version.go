/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
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
	Long: `This command is used to print the Odigos version.
Both the CLI version and the Odigos components in your cluster will be printed.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		odigosClusterVersion, clusterVersionErr := getOdigosVersionInCluster(cmd)

		cliFlag, _ := cmd.Flags().GetBool(cliFlag)
		clusterFlag, _ := cmd.Flags().GetBool(clusterFlag)

		// check for errors - both flags cannot be used at the same time
		if cliFlag && clusterFlag {
			return errors.New("Only one of the flags --cli or --cluster can be used at a time")
		}

		// handle case where only one of the flags is used, and print the version accordingly or return an error
		if cliFlag {
			fmt.Printf("%s\n", OdigosVersion)
			return nil
		} else if clusterFlag {
			if clusterVersionErr != nil {
				return clusterVersionErr
			}
			fmt.Printf("%s\n", odigosClusterVersion)
			return nil
		}

		fmt.Printf("Odigos CLI: \n  Version: %s\n  GitCommit: %s\n  BuildDate: %s\n\n", OdigosVersion, OdigosCommit, OdigosDate)
		var odigosClusterVersionText string
		if clusterVersionErr != nil {
			odigosClusterVersionText = fmt.Sprintf("Status: %s", clusterVersionErr.Error())
		} else {
			odigosClusterVersionText = fmt.Sprintf("Version: %s", odigosClusterVersion)
		}
		fmt.Printf("Odigos in Cluster:\n  %s\n", odigosClusterVersionText)
		return nil
	},
	Example: `
# Print the version of odigos CLI and Odigos deployment in your cluster
odigos version
`,
}

// returns the odigos version in the cluster (as v1.2.3) or err if the version cannot be determined
func getOdigosVersionInCluster(cmd *cobra.Command) (string, error) {

	// get the client which is generated from the root command
	ctx := cmd.Context()
	client, err := cmdcontext.KubeClientFromContext(ctx)
	if err != nil {
		return "", errors.New("No Kubernetes cluster found")
	}

	odigosns, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		if resources.IsErrNoOdigosNamespaceFound(err) {
			return "", errors.New("Not Installed")
		} else {
			return "", errors.New("Multiple Odigos installations found")
		}
	}

	v, err := getters.GetOdigosVersionInClusterFromConfigMap(ctx, client.Clientset, odigosns)

	return v, err
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().Bool(cliFlag, false, "prints only the CLI version")
	versionCmd.Flags().Bool(clusterFlag, false, "prints only the version of odigos deployed in the cluster")
}
